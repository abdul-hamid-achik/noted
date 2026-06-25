package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Version is a single immutable snapshot of a note, persisted in the vault so history survives a
// SQLite rebuild (the index is rebuildable; the vault is the source of truth — including history).
type Version struct {
	NoteID        int64
	VersionNumber int64
	Title         string
	Created       time.Time
	Content       string
}

type versionFrontmatter struct {
	NoteID  int64     `yaml:"note_id"`
	Version int64     `yaml:"version"`
	Title   string    `yaml:"title"`
	Created time.Time `yaml:"created,omitempty"`
}

// versionsDir is the hidden directory that holds version snapshots. It lives under the vault but is
// ignored by List (which skips dot-dirs / non-.md top-level entries), so versions never appear as notes.
func (v *Vault) versionsDir() string { return filepath.Join(v.dir, ".noted", "versions") }

// WriteVersion persists one snapshot to .noted/versions/<noteID>/<version>.md. Snapshots are
// immutable: an existing file is left untouched and (false, nil) is returned. Returns true when a new
// file is written.
func (v *Vault) WriteVersion(ver Version) (bool, error) {
	if ver.NoteID <= 0 || ver.VersionNumber <= 0 {
		return false, fmt.Errorf("vault: version needs positive note id and version number")
	}
	dir := filepath.Join(v.versionsDir(), strconv.FormatInt(ver.NoteID, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	path := filepath.Join(dir, strconv.FormatInt(ver.VersionNumber, 10)+".md")
	if _, err := os.Stat(path); err == nil {
		return false, nil // already persisted
	}
	fm := versionFrontmatter{NoteID: ver.NoteID, Version: ver.VersionNumber, Title: ver.Title, Created: ver.Created}
	y, err := yaml.Marshal(fm)
	if err != nil {
		return false, err
	}
	body := "---\n" + string(y) + "---\n\n" + ver.Content + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteVersions removes all persisted snapshots for a note id. Call this when a note is deleted so
// stale history can't later graft onto a different note that reuses the id. A no-op if none exist.
func (v *Vault) DeleteVersions(id int64) error {
	if id <= 0 {
		return nil
	}
	err := os.RemoveAll(filepath.Join(v.versionsDir(), strconv.FormatInt(id, 10)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// AllVersions reads every persisted snapshot, sorted by (note id, version number).
func (v *Vault) AllVersions() ([]Version, error) {
	root := v.versionsDir()
	noteDirs, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Version
	for _, nd := range noteDirs {
		if !nd.IsDir() {
			continue
		}
		sub := filepath.Join(root, nd.Name())
		files, err := os.ReadDir(sub)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
				continue
			}
			ver, err := parseVersion(filepath.Join(sub, f.Name()))
			if err != nil {
				continue // skip unreadable snapshots rather than failing the whole scan
			}
			out = append(out, ver)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NoteID != out[j].NoteID {
			return out[i].NoteID < out[j].NoteID
		}
		return out[i].VersionNumber < out[j].VersionNumber
	})
	return out, nil
}

func parseVersion(path string) (Version, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Version{}, err
	}
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return Version{}, fmt.Errorf("vault: version %s has no frontmatter", path)
	}
	rest := s[len("---\n"):]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return Version{}, fmt.Errorf("vault: version %s has unterminated frontmatter", path)
	}
	var fm versionFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:idx]), &fm); err != nil {
		return Version{}, err
	}
	body := strings.Trim(rest[idx+len("\n---"):], "\n")
	return Version{
		NoteID:        fm.NoteID,
		VersionNumber: fm.Version,
		Title:         fm.Title,
		Created:       fm.Created,
		Content:       body,
	}, nil
}
