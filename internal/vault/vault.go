// Package vault stores notes as Markdown files with YAML frontmatter in a directory (the "vault").
// It is the source of truth for noted; SQLite is a rebuildable index over it. This package is pure
// file I/O + (de)serialization — no database, no TUI.
package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// now is indirected so tests can pin timestamps.
var now = time.Now

// Note is a single vault note (distinct from db.Note).
type Note struct {
	ID      int64  // optional stable id (mirrors the SQLite index; 0 if unindexed)
	Path    string // filesystem path (set on Read/Write)
	Title   string
	Tags    []string
	Folder  string // folder name ("" = none)
	Pinned  bool
	Created time.Time
	Updated time.Time
	Content string // markdown body (without frontmatter)
}

// Vault is a directory of Markdown notes.
type Vault struct{ dir string }

// Open ensures the vault directory exists and returns a handle.
func Open(dir string) (*Vault, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("vault: empty directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("vault: create %q: %w", dir, err)
	}
	return &Vault{dir: dir}, nil
}

// Dir returns the vault's root directory.
func (v *Vault) Dir() string { return v.dir }

type frontmatter struct {
	ID      int64     `yaml:"id,omitempty"`
	Title   string    `yaml:"title"`
	Tags    []string  `yaml:"tags,omitempty"`
	Folder  string    `yaml:"folder,omitempty"`
	Pinned  bool      `yaml:"pinned,omitempty"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
}

// Serialize renders a note to its on-disk Markdown+frontmatter form.
func Serialize(n Note) string {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	_ = enc.Encode(frontmatter{
		ID: n.ID, Title: n.Title, Tags: n.Tags, Folder: n.Folder, Pinned: n.Pinned,
		Created: n.Created, Updated: n.Updated,
	})
	_ = enc.Close()
	buf.WriteString("---\n\n")
	buf.WriteString(n.Content)
	if !strings.HasSuffix(n.Content, "\n") {
		buf.WriteString("\n")
	}
	return buf.String()
}

// Parse reads a note from its on-disk form. Files without frontmatter are treated as pure content.
func Parse(data []byte) (Note, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	var n Note
	if strings.HasPrefix(s, "---\n") {
		rest := s[len("---\n"):]
		if idx := strings.Index(rest, "\n---"); idx >= 0 {
			fmText := rest[:idx]
			body := rest[idx+len("\n---"):]
			var fm frontmatter
			if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
				// Not real frontmatter (e.g. a note that opens with a thematic break, or malformed
				// YAML). Treat the whole file as content so the note is never dropped on rebuild.
				n.Content = strings.Trim(s, "\n")
				return n, nil
			}
			n.ID = fm.ID
			n.Title, n.Tags, n.Folder, n.Pinned = fm.Title, fm.Tags, fm.Folder, fm.Pinned
			n.Created, n.Updated = fm.Created, fm.Updated
			// Trim the blank line(s) bracketing the body (the newline ending the closing "---" line
			// and the trailing newline Serialize always writes), so content round-trips cleanly.
			n.Content = strings.Trim(body, "\n")
			return n, nil
		}
	}
	n.Content = s
	return n, nil
}

// Read loads a single note by path (absolute, or relative to the vault dir).
func (v *Vault) Read(path string) (Note, error) {
	p := v.resolve(path)
	data, err := os.ReadFile(p)
	if err != nil {
		return Note{}, err
	}
	n, err := Parse(data)
	if err != nil {
		return Note{}, err
	}
	n.Path = p
	if strings.TrimSpace(n.Title) == "" {
		n.Title = titleFromPath(p)
	}
	return n, nil
}

// List scans the vault for *.md notes, newest (by Updated) first.
func (v *Vault) List() ([]Note, error) {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		n, err := v.Read(filepath.Join(v.dir, e.Name()))
		if err != nil {
			continue // skip unreadable files rather than failing the whole scan
		}
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Updated.After(notes[j].Updated) })
	return notes, nil
}

// Write persists a note. New notes (empty Path) get a unique slug-based filename. Updated is always
// set to now; Created is set to now only if unset.
func (v *Vault) Write(n Note) (Note, error) {
	if n.Created.IsZero() {
		n.Created = now()
	}
	n.Updated = now()
	if n.Path == "" {
		n.Path = v.uniquePath(n.Title)
	} else {
		n.Path = v.resolve(n.Path)
	}
	if err := os.WriteFile(n.Path, []byte(Serialize(n)), 0o644); err != nil {
		return Note{}, err
	}
	return n, nil
}

// WriteRaw writes a note exactly as given (no timestamp defaults) — used by export/import where the
// timestamps and id come from the source of record. Assigns a unique slug path when Path is empty.
func (v *Vault) WriteRaw(n Note) (Note, error) {
	if n.Path == "" {
		n.Path = v.uniquePath(n.Title)
	} else {
		n.Path = v.resolve(n.Path)
	}
	if err := os.WriteFile(n.Path, []byte(Serialize(n)), 0o644); err != nil {
		return Note{}, err
	}
	return n, nil
}

// Sync writes a note through to the vault, reusing the existing file for its id when present (so a
// title change updates one file rather than orphaning the old one). New notes get a unique slug path.
func (v *Vault) Sync(n Note) (Note, error) {
	if n.ID > 0 {
		if existing, ok := v.findByID(n.ID); ok {
			n.Path = existing.Path
		}
	}
	return v.WriteRaw(n)
}

// DeleteByID removes the vault file backing a note id (no-op if absent).
func (v *Vault) DeleteByID(id int64) error {
	if n, ok := v.findByID(id); ok {
		return v.Delete(n.Path)
	}
	return nil
}

func (v *Vault) findByID(id int64) (Note, bool) {
	notes, err := v.List()
	if err != nil {
		return Note{}, false
	}
	for _, n := range notes {
		if n.ID == id {
			return n, true
		}
	}
	return Note{}, false
}

// Delete removes a note file.
func (v *Vault) Delete(path string) error {
	return os.Remove(v.resolve(path))
}

func (v *Vault) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(v.dir, path)
}

func (v *Vault) uniquePath(title string) string {
	base := Slugify(title)
	p := filepath.Join(v.dir, base+".md")
	for i := 2; fileExists(p); i++ {
		p = filepath.Join(v.dir, fmt.Sprintf("%s-%d.md", base, i))
	}
	return p
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a title into a filesystem-safe base name.
func Slugify(title string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

func titleFromPath(p string) string {
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	base = strings.ReplaceAll(base, "-", " ")
	return strings.TrimSpace(base)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
