package tui

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/harmonica"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/styles"
)

// View names
type ViewName string

type EditorMode string

const (
	ViewNotes     ViewName = "notes"
	ViewEditor    ViewName = "editor"
	ViewSearch    ViewName = "search"
	ViewTags      ViewName = "tags"
	ViewDashboard ViewName = "dashboard"
	ViewDaily     ViewName = "daily"
	ViewTasks     ViewName = "tasks"
	ViewBacklinks ViewName = "backlinks"
	ViewFolders   ViewName = "folders"
	ViewHelp      ViewName = "help"
	ViewSettings  ViewName = "settings"
	ViewGraph     ViewName = "graph"
	ViewHistory   ViewName = "history"
	ViewPreview   ViewName = "preview"
	ViewTemplates ViewName = "templates"
)

const (
	EditorModeSplit   EditorMode = "split"
	EditorModeEdit    EditorMode = "edit"
	EditorModePreview EditorMode = "preview"
)

// Style aliases for backward compatibility
var (
	primary = styles.Primary
	accent  = styles.Accent

	background = styles.Background
	surface    = styles.Surface
	surfaceAlt = styles.SurfaceAlt
	border     = styles.Border
	textCol    = styles.Text
	mutedText  = styles.MutedText
)

// Reusable style objects
var (
	titleStyle      = styles.TitleStyle
	headingStyle    = styles.HeadingStyle
	subheadingStyle = styles.SubheadingStyle
	mutedStyle      = styles.MutedStyle

	inputStyle        = styles.InputStyle
	inputFocusedStyle = styles.InputFocusedStyle

	panelStyle = styles.PanelStyle
	cardStyle  = styles.CardStyle
	modalStyle = styles.ModalStyle

	statusStyle = styles.StatusStyle
)

// Main application model
type Model struct {
	db  *db.Queries
	ctx context.Context

	currentView  ViewName
	previousView ViewName
	width        int
	height       int

	notes       []db.Note
	noteTags    map[int64][]db.Tag
	currentNote *db.Note

	tags      []db.Tag
	tagCounts map[int64]int

	folders []db.Folder

	searchQuery   string
	searchResults []db.Note

	backlinks []db.Note

	tasks []Task

	versions []db.NoteVersion

	templates []db.Template

	loading               bool
	err                   error
	quitting              bool
	showHelp              bool
	componentsInitialized bool

	noteList     list.Model
	tagList      list.Model
	folderTree   list.Model
	taskList     list.Model
	versionList  list.Model
	templateList list.Model
	searchInput  textinput.Model
	editorInput  textinput.Model
	contentArea  textarea.Model
	previewVP    viewport.Model
	modeBar      progress.Model
	contentBar   progress.Model
	previewBar   progress.Model

	spinner spinner.Model

	// Editor form controls
	editorTitleField *huh.Input
	editorModeField  *huh.Select[EditorMode]

	// Navigation state for vim-style (g prefix)
	pendingKey string

	// Editor state
	isCreating    bool
	isPinned      bool
	editorTitle   string
	editorContent string
	editorDirty   bool
	editorFocus   string // "title", "content", or "preview"
	editorMode    EditorMode

	// Toast notifications
	toastMessage string
	toastTimeout int
	toastTimer   *time.Timer

	stats           DashboardStats
	mouseHandler    *MouseHandler
	editorBackArmed bool
}

type hitRect struct {
	X int
	Y int
	W int
	H int
}

func (r hitRect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

type editorActionHit struct {
	Name string
	Rect hitRect
}

type editorLayoutMetrics struct {
	Sidebar hitRect
	Editor  hitRect
	Mode    hitRect
	Title   hitRect
	Content hitRect
	Preview hitRect
	Actions []editorActionHit
}

func (m editorLayoutMetrics) ActionAt(x, y int) (string, bool) {
	for _, action := range m.Actions {
		if action.Rect.Contains(x, y) {
			return action.Name, true
		}
	}
	return "", false
}

type DashboardStats struct {
	TotalNotes   int64
	TotalTags    int64
	TodayNotes   int64
	WeekNotes    int64
	PinnedNotes  int64
	ExpiredNotes int64
	TotalFolders int64
}

type Task struct {
	ID        int64
	NoteID    int64
	NoteTitle string
	Content   string
	Completed bool
	Line      int
}

// Note list item
type NoteItem struct {
	note db.Note
	tags []db.Tag
}

func (i NoteItem) Title() string {
	if i.note.Pinned.Valid && i.note.Pinned.Bool {
		return "📌 " + i.note.Title
	}
	return i.note.Title
}
func (i NoteItem) Description() string {
	preview := i.note.Content
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	preview = strings.ReplaceAll(preview, "\n", " ")
	return preview
}
func (i NoteItem) FilterValue() string {
	tags := ""
	for _, t := range i.tags {
		tags += t.Name + " "
	}
	return i.note.Title + " " + i.note.Content + " " + tags
}

// Tag list item
type TagItem struct {
	name      string
	noteCount int64
}

func (i TagItem) Title() string       { return "#" + i.name }
func (i TagItem) Description() string { return fmt.Sprintf("%d notes", i.noteCount) }
func (i TagItem) FilterValue() string { return i.name }

// Folder list item
type FolderItem struct {
	folder    db.Folder
	noteCount int
}

func (i FolderItem) Title() string       { return i.folder.Name }
func (i FolderItem) Description() string { return fmt.Sprintf("%d notes", i.noteCount) }
func (i FolderItem) FilterValue() string { return i.folder.Name }

// Task list item
type TaskItem struct {
	task Task
}

func (i TaskItem) Title() string {
	checkbox := "[ ]"
	if i.task.Completed {
		checkbox = "[x]"
	}
	return fmt.Sprintf("%s %s", checkbox, i.task.Content)
}
func (i TaskItem) Description() string { return "From: " + i.task.NoteTitle }
func (i TaskItem) FilterValue() string { return i.task.Content + " " + i.task.NoteTitle }

// Version list item
type VersionItem struct {
	version db.NoteVersion
}

func (i VersionItem) Title() string {
	return fmt.Sprintf("Version %d", i.version.VersionNumber)
}
func (i VersionItem) Description() string {
	if i.version.CreatedAt.Valid {
		return i.version.CreatedAt.Time.Format("2006-01-02 15:04")
	}
	return ""
}
func (i VersionItem) FilterValue() string { return "" }

// Template list item
type TemplateItem struct {
	template db.Template
}

func (i TemplateItem) Title() string { return i.template.Name }
func (i TemplateItem) Description() string {
	if i.template.CreatedAt.Valid {
		return fmt.Sprintf("Created: %s", i.template.CreatedAt.Time.Format("2006-01-02"))
	}
	return ""
}
func (i TemplateItem) FilterValue() string { return i.template.Name + " " + i.template.Content }

type (
	notesLoadedMsg     struct{ notes []db.Note }
	tagsLoadedMsg      struct{ tags []db.GetTagsWithCountRow }
	foldersLoadedMsg   struct{ folders []db.Folder }
	noteSavedMsg       struct{ note db.Note }
	noteDeletedMsg     struct{}
	searchResultsMsg   struct{ results []db.Note }
	backlinksLoadedMsg struct{ backlinks []db.Note }
	tasksLoadedMsg     struct{ tasks []Task }
	statsLoadedMsg     struct{ stats DashboardStats }
	versionsLoadedMsg  struct{ versions []db.NoteVersion }
	templatesLoadedMsg struct{ templates []db.Template }
	toastMsg           struct{ message string }
)

func New(ctx context.Context, database *db.Queries) (*tea.Program, error) {
	m := NewModel(database, ctx)

	p := tea.NewProgram(m)

	return p, nil
}

func NewModel(database *db.Queries, ctx context.Context) Model {
	return Model{
		db:           database,
		ctx:          ctx,
		width:        80,
		height:       24,
		currentView:  ViewNotes,
		editorMode:   EditorModeSplit,
		noteTags:     make(map[int64][]db.Tag),
		tagCounts:    make(map[int64]int),
		loading:      true,
		mouseHandler: NewMouseHandler(),
	}
}

func (m *Model) initComponents() {
	// Use terminal dimensions (will be set by WindowSizeMsg)
	// Initialize with reasonable defaults that will be updated on resize
	contentWidth := m.width - 40
	contentHeight := m.height - 6

	delegate := list.NewDefaultDelegate()

	// Notes list - main content
	m.noteList = list.New(nil, delegate, contentWidth, contentHeight)
	m.noteList.Title = "Notes"
	m.noteList.SetShowStatusBar(true)
	m.noteList.SetFilteringEnabled(true)
	m.noteList.SetShowHelp(false)
	m.noteList.KeyMap.CursorUp = key.NewBinding(key.WithKeys("k", "up", "ctrl+p"))
	m.noteList.KeyMap.CursorDown = key.NewBinding(key.WithKeys("j", "down", "ctrl+n"))
	m.noteList.KeyMap.Filter = key.NewBinding(key.WithKeys("/"))
	m.noteList.KeyMap.ClearFilter = key.NewBinding(key.WithKeys("esc"))
	m.noteList.KeyMap.ShowFullHelp = key.NewBinding(key.WithKeys("?"))
	m.noteList.KeyMap.Quit = key.NewBinding(key.WithKeys("q", "ctrl+c"))

	// Tags list
	m.tagList = list.New(nil, delegate, contentWidth, contentHeight)
	m.tagList.Title = "Tags"
	m.tagList.SetShowStatusBar(false)
	m.tagList.SetFilteringEnabled(true)
	m.tagList.SetShowHelp(false)

	// Folders list
	m.folderTree = list.New(nil, delegate, contentWidth, contentHeight)
	m.folderTree.Title = "Folders"
	m.folderTree.SetShowStatusBar(false)
	m.folderTree.SetFilteringEnabled(true)
	m.folderTree.SetShowHelp(false)

	// Tasks list
	m.taskList = list.New(nil, delegate, contentWidth, contentHeight)
	m.taskList.Title = "Tasks"
	m.taskList.SetShowStatusBar(true)
	m.taskList.SetFilteringEnabled(true)
	m.taskList.SetShowHelp(false)

	// Versions list
	m.versionList = list.New(nil, delegate, contentWidth, contentHeight)
	m.versionList.Title = "Version History"
	m.versionList.SetShowStatusBar(false)
	m.versionList.SetShowHelp(false)

	// Templates list
	m.templateList = list.New(nil, delegate, contentWidth, contentHeight)
	m.templateList.Title = "Templates"
	m.templateList.SetShowStatusBar(false)
	m.templateList.SetFilteringEnabled(true)
	m.templateList.SetShowHelp(false)

	// Search input
	m.searchInput = textinput.New()
	m.searchInput.Placeholder = "Search notes..."
	m.searchInput.Prompt = "🔍 "
	m.searchInput.Focus()

	// Editor title
	m.editorInput = textinput.New()
	m.editorInput.Placeholder = "Note title..."
	m.editorInput.Prompt = "📝 "

	// Content textarea (editor)
	m.contentArea = textarea.New()
	m.contentArea.Placeholder = "Start writing your note..."
	m.contentArea.Prompt = ""
	m.contentArea.SetHeight(contentHeight - 8)
	m.contentArea.SetWidth(contentWidth - 2)
	m.contentArea.ShowLineNumbers = false
	m.contentArea.CharLimit = 0
	taStyles := m.contentArea.Styles()
	taStyles.Focused.Base = lipgloss.NewStyle().Foreground(textCol)
	taStyles.Focused.CursorLine = lipgloss.NewStyle().Background(surfaceAlt)
	taStyles.Focused.LineNumber = lipgloss.NewStyle().Foreground(mutedText)
	taStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(primary)
	taStyles.Blurred.Base = lipgloss.NewStyle().Foreground(textCol)
	taStyles.Blurred.LineNumber = lipgloss.NewStyle().Foreground(border)
	taStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(border)
	m.contentArea.SetStyles(taStyles)
	m.contentArea.Focus()

	// Preview viewport
	m.previewVP = viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	m.modeBar = progress.New(
		progress.WithWidth(12),
		progress.WithoutPercentage(),
		progress.WithFillCharacters(progress.DefaultFullCharFullBlock, progress.DefaultEmptyCharBlock),
	)
	m.contentBar = progress.New(
		progress.WithWidth(20),
		progress.WithoutPercentage(),
		progress.WithFillCharacters(progress.DefaultFullCharFullBlock, progress.DefaultEmptyCharBlock),
	)
	m.previewBar = progress.New(
		progress.WithWidth(20),
		progress.WithoutPercentage(),
		progress.WithFillCharacters(progress.DefaultFullCharFullBlock, progress.DefaultEmptyCharBlock),
	)

	// Spinner
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot

	m.initEditorFields()

	m.editorFocus = "content"
	m.syncEditorLayout()

}

func (m *Model) initEditorFields() {
	theme := huh.ThemeFunc(huh.ThemeCharm)

	titleField := huh.NewInput().
		Inline(true).
		Prompt("title> ").
		Placeholder("Untitled").
		Value(&m.editorTitle).
		CharLimit(200)
	titleField.WithTheme(theme)
	m.editorTitleField = titleField

	modeField := huh.NewSelect[EditorMode]().
		Inline(true).
		Title("mode").
		Options(
			huh.NewOption("split", EditorModeSplit),
			huh.NewOption("edit", EditorModeEdit),
		).
		Value(&m.editorMode)
	modeField.WithTheme(theme)
	m.editorModeField = modeField
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadNotesCmd(),
		m.loadTagsCmd(),
		m.loadFoldersCmd(),
		m.loadTemplatesCmd(),
	)
}

func (m *Model) loadNotesCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		notes, err := m.db.ListNotes(m.ctx, db.ListNotesParams{Limit: 100, Offset: 0})
		if err != nil {
			return err
		}

		for i := range notes {
			tags, err := m.db.GetTagsForNote(m.ctx, notes[i].ID)
			if err != nil {
				continue
			}
			m.noteTags[notes[i].ID] = tags
		}

		return notesLoadedMsg{notes: notes}
	}
}

func (m *Model) loadTagsCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		tags, err := m.db.GetTagsWithCount(m.ctx)
		if err != nil {
			return err
		}
		return tagsLoadedMsg{tags: tags}
	}
}

func (m *Model) loadFoldersCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		folders, err := m.db.ListFolders(m.ctx)
		if err != nil {
			return err
		}
		return foldersLoadedMsg{folders: folders}
	}
}

func (m *Model) loadTemplatesCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		templates, err := m.db.ListTemplates(m.ctx)
		if err != nil {
			return err
		}
		return templatesLoadedMsg{templates: templates}
	}
}

func (m *Model) loadStatsCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		stats := DashboardStats{}

		count, _ := m.db.CountNotes(m.ctx)
		stats.TotalNotes = count

		count, _ = m.db.CountTags(m.ctx)
		stats.TotalTags = count

		notes, _ := m.db.GetPinnedNotes(m.ctx)
		stats.PinnedNotes = int64(len(notes))

		expired, _ := m.db.GetExpiredNotes(m.ctx)
		stats.ExpiredNotes = int64(len(expired))

		folders, _ := m.db.ListFolders(m.ctx)
		stats.TotalFolders = int64(len(folders))

		return statsLoadedMsg{stats: stats}
	}
}

func (m *Model) loadBacklinksCmd(noteID int64) tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		backlinks, err := m.db.GetBacklinks(m.ctx, noteID)
		if err != nil {
			return err
		}
		return backlinksLoadedMsg{backlinks: backlinks}
	}
}

func (m *Model) loadVersionsCmd(noteID int64) tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		versions, err := m.db.GetNoteVersions(m.ctx, noteID)
		if err != nil {
			return err
		}
		return versionsLoadedMsg{versions: versions}
	}
}

func (m *Model) searchNotesCmd(query string) tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		notes, err := m.db.SearchNotesByTitle(m.ctx, query)
		if err != nil {
			notes, err = m.db.SearchNotesContent(m.ctx, db.SearchNotesContentParams{Content: query, Title: query, Limit: 50})
			if err != nil {
				return err
			}
		}
		return searchResultsMsg{results: notes}
	}
}

func (m *Model) loadTasksCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		allNotes, err := m.db.GetAllNotes(m.ctx)
		if err != nil {
			return err
		}

		var tasks []Task
		for _, note := range allNotes {
			lines := strings.Split(note.Content, "\n")
			for lineNum, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "* [ ]") {
					tasks = append(tasks, Task{
						ID:        int64(len(tasks)),
						NoteID:    note.ID,
						NoteTitle: note.Title,
						Content:   strings.TrimPrefix(strings.TrimPrefix(line, "- [ ]"), "* [ ]"),
						Completed: false,
						Line:      lineNum + 1,
					})
				} else if strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "* [x]") {
					tasks = append(tasks, Task{
						ID:        int64(len(tasks)),
						NoteID:    note.ID,
						NoteTitle: note.Title,
						Content:   strings.TrimPrefix(strings.TrimPrefix(line, "- [x]"), "* [x]"),
						Completed: true,
						Line:      lineNum + 1,
					})
				}
			}
		}
		return tasksLoadedMsg{tasks: tasks}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Initialize components on first window size message if not already done
		if !m.componentsInitialized {
			m.initComponents()
			m.componentsInitialized = true
		}
		return m.handleResize(msg)

	case tea.MouseMsg:
		// Mouse events are passed to the current view for handling
		// Each view's update function will handle mouse clicks on their list components
		return m.handleMouseMsg(msg)

	case tea.QuitMsg:
		m.quitting = true
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case toastMsg:
		m.toastMessage = msg.message
		m.toastTimeout = 3 // 3 seconds
		// Start timer to clear toast
		if m.toastTimer != nil {
			m.toastTimer.Stop()
		}
		m.toastTimer = time.AfterFunc(3*time.Second, func() {
			m.toastMessage = ""
		})
		return m, nil

	case error:
		m.err = msg
		// Show error as toast
		m.toastMessage = "Error: " + msg.Error()
		m.toastTimeout = 5
		return m, nil

	case notesLoadedMsg:
		m.notes = msg.notes
		m.loading = false
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = NoteItem{note: n, tags: m.noteTags[n.ID]}
		}
		m.noteList.SetItems(items)
		return m, nil

	case tagsLoadedMsg:
		m.tags = make([]db.Tag, len(msg.tags))
		items := make([]list.Item, len(msg.tags))
		for i, t := range msg.tags {
			m.tags[i] = db.Tag{ID: t.ID, Name: t.Name}
			m.tagCounts[t.ID] = int(t.NoteCount)
			items[i] = TagItem{name: t.Name, noteCount: t.NoteCount}
		}
		m.tagList.SetItems(items)
		return m, nil

	case foldersLoadedMsg:
		m.folders = msg.folders
		items := make([]list.Item, len(msg.folders))
		for i, f := range msg.folders {
			noteCount := 0
			notes, _ := m.db.GetNotesByFolder(m.ctx, sql.NullInt64{Int64: f.ID, Valid: true})
			noteCount = len(notes)
			items[i] = FolderItem{folder: f, noteCount: noteCount}
		}
		m.folderTree.SetItems(items)
		return m, nil

	case templatesLoadedMsg:
		m.templates = msg.templates
		items := make([]list.Item, len(msg.templates))
		for i, t := range msg.templates {
			items[i] = TemplateItem{template: t}
		}
		m.templateList.SetItems(items)
		return m, nil

	case searchResultsMsg:
		m.searchResults = msg.results
		items := make([]list.Item, len(msg.results))
		for i, n := range msg.results {
			tags, _ := m.db.GetTagsForNote(m.ctx, n.ID)
			items[i] = NoteItem{note: n, tags: tags}
		}
		m.noteList.SetItems(items)
		return m, nil

	case backlinksLoadedMsg:
		m.backlinks = msg.backlinks
		return m, nil

	case tasksLoadedMsg:
		m.tasks = msg.tasks
		items := make([]list.Item, len(msg.tasks))
		for i, t := range msg.tasks {
			items[i] = TaskItem{task: t}
		}
		m.taskList.SetItems(items)
		return m, nil

	case versionsLoadedMsg:
		m.versions = msg.versions
		items := make([]list.Item, len(msg.versions))
		for i, v := range msg.versions {
			items[i] = VersionItem{version: v}
		}
		m.versionList.SetItems(items)
		return m, nil

	case statsLoadedMsg:
		m.stats = msg.stats
		return m, nil

	case noteSavedMsg:
		m.currentNote = &msg.note
		m.editorDirty = false
		return m, m.loadNotesCmd()

	case noteDeletedMsg:
		m.currentNote = nil
		return m, m.loadNotesCmd()
	}

	// Global key handling
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "?" && !m.showHelp {
			m.previousView = m.currentView
			m.currentView = ViewHelp
			return m, nil
		}

		if msg.String() == "esc" {
			if m.showHelp {
				m.currentView = m.previousView
				m.showHelp = false
				return m, nil
			}
			if m.currentView == ViewPreview {
				m.currentView = ViewEditor
				m.editorMode = EditorModeSplit
				m.syncEditorLayout()
				return m, nil
			}
			if m.currentView == ViewEditor {
				if m.editorDirty && !m.editorBackArmed {
					m.editorBackArmed = true
					return m, func() tea.Msg { return toastMsg{message: "Unsaved changes. Press Esc again to discard."} }
				}
				m.currentView = ViewNotes
				m.currentNote = nil
				m.editorBackArmed = false
				return m, nil
			}
			if m.currentView == ViewSearch {
				m.searchQuery = ""
				m.currentView = ViewNotes
				return m, m.loadNotesCmd()
			}
			if m.currentView != ViewNotes {
				m.currentView = ViewNotes
				return m, nil
			}
		}

		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Vim-style navigation: g prefix
		if m.currentView != ViewEditor && m.currentView != ViewSearch && m.currentView != ViewPreview {
			// Check for pending g key
			if m.pendingKey == "g" {
				m.pendingKey = ""
				switch msg.String() {
				case "h": // Back / left
					m.currentView = ViewNotes
				case "j": // Down / next view
					m.currentView = ViewTags
				case "k": // Up / previous view
					m.currentView = ViewSettings
				case "l": // Forward / right
					m.currentView = ViewSearch
				case "g": // gg = go to first (Notes)
					m.currentView = ViewNotes
				}
				return m, nil
			}

			// Start vim prefix
			switch msg.String() {
			case "g":
				m.pendingKey = "g"
			}
		}
	}

	switch m.currentView {
	case ViewNotes:
		return m.updateNotesView(msg)
	case ViewEditor:
		return m.updateEditorView(msg)
	case ViewPreview:
		return m.updatePreviewView(msg)
	case ViewSearch:
		return m.updateSearchView(msg)
	case ViewTags:
		return m.updateTagsView(msg)
	case ViewDashboard:
		return m.updateDashboardView(msg)
	case ViewDaily:
		return m.updateDailyView(msg)
	case ViewTasks:
		return m.updateTasksView(msg)
	case ViewBacklinks:
		return m.updateBacklinksView(msg)
	case ViewFolders:
		return m.updateFoldersView(msg)
	case ViewHelp:
		return m.updateHelpView(msg)
	case ViewSettings:
		return m.updateSettingsView(msg)
	case ViewGraph:
		return m.updateGraphView(msg)
	case ViewHistory:
		return m.updateHistoryView(msg)
	case ViewTemplates:
		return m.updateTemplatesView(msg)
	}

	return m, cmd
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	if m.mouseHandler != nil {
		m.mouseHandler.SetSize(m.width, m.height)
	}

	sidebarWidth := 35
	contentWidth := m.width - sidebarWidth - 1
	contentHeight := m.height - 6

	// Main list
	m.noteList.SetSize(contentWidth, contentHeight)
	m.tagList.SetSize(contentWidth, contentHeight)
	m.folderTree.SetSize(contentWidth, contentHeight)
	m.taskList.SetSize(contentWidth, contentHeight)
	m.versionList.SetSize(contentWidth, contentHeight)
	m.templateList.SetSize(contentWidth, contentHeight)

	// Search input
	m.searchInput.SetWidth(sidebarWidth - 4)
	m.syncEditorLayout()

	return m, nil
}

func (m *Model) setEditorFocus(focus string) {
	switch focus {
	case "title", "mode", "content", "preview":
		m.editorFocus = focus
	default:
		m.editorFocus = "content"
	}

	if m.editorTitleField != nil {
		if m.editorFocus == "title" {
			_ = m.editorTitleField.Focus()
		} else {
			_ = m.editorTitleField.Blur()
		}
	}

	if m.editorModeField != nil {
		if m.editorFocus == "mode" {
			_ = m.editorModeField.Focus()
		} else {
			_ = m.editorModeField.Blur()
		}
	}

	if m.editorFocus == "content" {
		m.contentArea.Focus()
	} else {
		m.contentArea.Blur()
	}
}

func (m *Model) openEditor(title, content string, focus string) {
	m.editorTitle = title
	m.editorContent = content
	m.editorDirty = false
	m.editorBackArmed = false
	m.editorMode = EditorModeSplit
	m.initEditorFields()
	m.contentArea.SetValue(content)
	m.syncEditorLayout()
	m.previewVP.SetContent(m.renderMarkdown(content, m.previewVP.Width()))
	m.setEditorFocus(focus)
	m.currentView = ViewEditor
}

func (m *Model) syncEditorLayout() {
	metrics := m.editorLayoutMetrics()
	if m.editorTitleField != nil {
		m.editorTitleField.WithWidth(max(8, metrics.Title.W))
	}
	if m.editorModeField != nil {
		m.editorModeField.WithWidth(max(8, metrics.Mode.W))
	}
	m.contentArea.SetWidth(max(8, metrics.Content.W-2))
	m.contentArea.SetHeight(max(4, metrics.Content.H-2))
	m.previewVP.SetWidth(max(8, metrics.Preview.W-2))
	m.previewVP.SetHeight(max(4, metrics.Preview.H-2))
	m.modeBar.SetWidth(max(8, min(16, metrics.Mode.W/3)))
	m.contentBar.SetWidth(max(8, metrics.Content.W-14))
	m.previewBar.SetWidth(max(8, metrics.Preview.W-14))
}

func (m Model) editorLayoutMetrics() editorLayoutMetrics {
	width := max(40, m.width)
	height := max(12, m.height)

	mainHeight := max(8, height-2)
	sidebarW := 32
	if width < 100 {
		sidebarW = max(24, width/4)
	}
	if sidebarW > width-26 {
		sidebarW = max(18, width-26)
	}

	metrics := editorLayoutMetrics{
		Sidebar: hitRect{X: 0, Y: 0, W: sidebarW, H: mainHeight},
		Editor:  hitRect{X: sidebarW, Y: 0, W: max(20, width-sidebarW), H: mainHeight},
	}

	innerX := metrics.Editor.X
	innerY := metrics.Editor.Y
	innerW := max(18, metrics.Editor.W)
	innerH := max(7, metrics.Editor.H)

	headerY := innerY
	modeY := innerY + 1
	titleY := innerY + 2
	bodyTop := innerY + 4
	bodyH := max(3, innerH-4)

	metrics.Mode = hitRect{X: innerX + 6, Y: modeY, W: max(10, innerW-6), H: 1}
	metrics.Title = hitRect{X: innerX + 6, Y: titleY, W: max(10, innerW-6), H: 1}

	paneH := max(3, bodyH-1)
	if m.editorMode == EditorModeSplit {
		const gap = 2
		contentW := max(10, (innerW-gap)/2)
		previewW := max(10, innerW-contentW-gap)
		metrics.Content = hitRect{X: innerX, Y: bodyTop + 1, W: contentW, H: paneH}
		metrics.Preview = hitRect{X: innerX + contentW + gap, Y: bodyTop + 1, W: previewW, H: paneH}
	} else {
		metrics.Content = hitRect{X: innerX, Y: bodyTop + 1, W: innerW, H: paneH}
		metrics.Preview = metrics.Content
	}

	// Top editor controls (clickable)
	tabX := innerX + 7
	topTabs := []editorActionHit{
		{Name: "focus_mode", Rect: hitRect{X: tabX, Y: headerY, W: len("[mode]"), H: 1}},
		{Name: "focus_title", Rect: hitRect{X: tabX + len("[mode]") + 1, Y: headerY, W: len("[title]"), H: 1}},
		{Name: "focus_content", Rect: hitRect{X: tabX + len("[mode]") + len("[title]") + 2, Y: headerY, W: len("[content]"), H: 1}},
	}
	metrics.Actions = append(metrics.Actions, topTabs...)
	if m.editorMode == EditorModeSplit {
		previewX := tabX + len("[mode]") + len("[title]") + len("[content]") + 3
		metrics.Actions = append(metrics.Actions, editorActionHit{Name: "focus_preview", Rect: hitRect{X: previewX, Y: headerY, W: len("[preview]"), H: 1}})
	}

	// Sidebar actions (2-line hit targets)
	sidebarContentX := metrics.Sidebar.X
	sidebarContentW := max(8, metrics.Sidebar.W)
	sidebarTop := metrics.Sidebar.Y
	metrics.Actions = append(metrics.Actions,
		editorActionHit{Name: "save", Rect: hitRect{X: sidebarContentX, Y: sidebarTop + 1, W: sidebarContentW, H: 1}},
		editorActionHit{Name: "toggle_mode", Rect: hitRect{X: sidebarContentX, Y: sidebarTop + 2, W: sidebarContentW, H: 1}},
		editorActionHit{Name: "full_preview", Rect: hitRect{X: sidebarContentX, Y: sidebarTop + 3, W: sidebarContentW, H: 1}},
		editorActionHit{Name: "focus", Rect: hitRect{X: sidebarContentX, Y: sidebarTop + 4, W: sidebarContentW, H: 1}},
		editorActionHit{Name: "back", Rect: hitRect{X: sidebarContentX, Y: sidebarTop + 5, W: sidebarContentW, H: 1}},
	)

	return metrics
}

func (m *Model) renderMarkdown(content string, width int) string {
	renderWidth := width
	if renderWidth < 20 {
		renderWidth = 20
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return content
	}

	preview, err := r.Render(content)
	if err != nil || strings.TrimSpace(preview) == "" {
		return content
	}

	return preview
}

// handleMouseMsg handles mouse events globally
func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Delegate all mouse handling to the mouse handler
	if m.mouseHandler != nil {
		return m.mouseHandler.Handle(msg, m)
	}
	return m, nil
}

func (m Model) updateNotesView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Don't process if components not yet initialized
	if !m.componentsInitialized {
		return m, nil
	}

	// Handle mouse wheel events
	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.noteList.Index() > 0 {
				m.noteList.CursorUp()
			}
		} else {
			if m.noteList.Index() < len(m.noteList.VisibleItems())-1 {
				m.noteList.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) { // Left click
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.noteList.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.noteList.VisibleItems()) {
					m.noteList.Select(itemIndex)
					if i, ok := m.noteList.SelectedItem().(NoteItem); ok {
						note, err := m.db.GetNote(m.ctx, i.note.ID)
						if err == nil {
							m.currentNote = &note
							m.isCreating = false
							m.isPinned = note.Pinned.Valid && note.Pinned.Bool
							m.openEditor(note.Title, note.Content, "content")
						}
					}
				}
			}
			return m, nil
		}
	}

	m.noteList, cmd = m.noteList.Update(msg)

	// Also update the search input in sidebar (for live filtering)
	m.searchInput, _ = m.searchInput.Update(msg)

	// If search input has content and user presses enter, perform search
	if m.searchInput.Value() != "" {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			query := m.searchInput.Value()
			m.searchQuery = query
			m.searchInput.Reset()
			m.currentView = ViewNotes
			return m, m.searchNotesCmd(query)
		}
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter", "right":
			if i, ok := m.noteList.SelectedItem().(NoteItem); ok {
				note, err := m.db.GetNote(m.ctx, i.note.ID)
				if err == nil {
					m.currentNote = &note
					m.isCreating = false
					m.isPinned = note.Pinned.Valid && note.Pinned.Bool
					m.openEditor(note.Title, note.Content, "content")
				}
			}
		case "n":
			m.isCreating = true
			m.editorDirty = false
			m.isPinned = false
			m.currentNote = nil
			m.openEditor("", "", "title")
		case "d":
			if i, ok := m.noteList.SelectedItem().(NoteItem); ok {
				err := m.db.DeleteNote(m.ctx, i.note.ID)
				if err == nil {
					return m, m.loadNotesCmd()
				}
			}
		case "p":
			// Pin/unpin
			if i, ok := m.noteList.SelectedItem().(NoteItem); ok {
				if i.note.Pinned.Valid && i.note.Pinned.Bool {
					_ = m.db.UnpinNote(m.ctx, i.note.ID)
				} else {
					_ = m.db.PinNote(m.ctx, i.note.ID)
				}
				return m, m.loadNotesCmd()
			}
		case "/":
			m.currentView = ViewSearch
			m.searchInput.Focus()
		case "t":
			m.currentView = ViewTags
		case "D":
			m.currentView = ViewDashboard
			return m, m.loadStatsCmd()
		case "l":
			m.currentView = ViewDaily
			return m, m.loadDailyNoteCmd()
		case "T":
			m.currentView = ViewTasks
			return m, m.loadTasksCmd()
		case "b":
			m.currentView = ViewBacklinks
			if m.currentNote != nil {
				return m, m.loadBacklinksCmd(m.currentNote.ID)
			}
		case "f":
			m.currentView = ViewFolders
		case ",":
			m.currentView = ViewSettings
		case "G":
			m.currentView = ViewGraph
		case "H":
			if m.currentNote != nil {
				m.currentView = ViewHistory
				return m, m.loadVersionsCmd(m.currentNote.ID)
			}
		case "e":
			if i, ok := m.noteList.SelectedItem().(NoteItem); ok {
				note, err := m.db.GetNote(m.ctx, i.note.ID)
				if err == nil {
					m.currentNote = &note
					m.isCreating = false
					m.isPinned = note.Pinned.Valid && note.Pinned.Bool
					m.openEditor(note.Title, note.Content, "content")
				}
			}
		case "v":
			m.currentView = ViewTemplates
		}
	}

	return m, cmd
}

func (m Model) updateEditorView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	switch m.editorFocus {
	case "title":
		previous := m.editorTitle
		if m.editorTitleField != nil {
			updated, c := m.editorTitleField.Update(msg)
			cmd = c
			if field, ok := updated.(*huh.Input); ok {
				m.editorTitleField = field
			}
		}
		if m.editorTitle != previous {
			m.editorDirty = true
			m.editorBackArmed = false
		}
	case "mode":
		previous := m.editorMode
		if m.editorModeField != nil {
			updated, c := m.editorModeField.Update(msg)
			cmd = c
			if field, ok := updated.(*huh.Select[EditorMode]); ok {
				m.editorModeField = field
			}
		}
		if m.editorMode != previous {
			m.editorBackArmed = false
			m.syncEditorLayout()
			if m.editorMode == EditorModeEdit && m.editorFocus == "preview" {
				m.setEditorFocus("content")
			}
		}
	case "content":
		previous := m.editorContent
		m.contentArea, cmd = m.contentArea.Update(msg)
		m.editorContent = m.contentArea.Value()
		if m.editorContent != previous {
			m.editorDirty = true
			m.editorBackArmed = false
		}
	case "preview":
		m.previewVP, cmd = m.previewVP.Update(msg)
	}

	// Handle keyboard shortcuts
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+s", "cmd+s":
			m.editorBackArmed = false
			return m, m.saveNoteCmd()
		case "ctrl+p":
			if m.editorMode == EditorModeSplit {
				m.editorMode = EditorModeEdit
			} else {
				m.editorMode = EditorModeSplit
			}
			m.syncEditorLayout()
			if m.editorMode == EditorModeEdit && m.editorFocus == "preview" {
				m.setEditorFocus("content")
			}
		case "p":
			if m.editorFocus == "content" {
				break
			}
			m.editorMode = EditorModePreview
			m.syncEditorLayout()
			m.currentView = ViewPreview
		case "tab":
			if m.editorMode == EditorModeSplit {
				switch m.editorFocus {
				case "mode":
					m.setEditorFocus("title")
				case "title":
					m.setEditorFocus("content")
				case "content":
					m.setEditorFocus("preview")
				default:
					m.setEditorFocus("mode")
				}
			} else {
				switch m.editorFocus {
				case "mode":
					m.setEditorFocus("title")
				case "title":
					m.setEditorFocus("content")
				default:
					m.setEditorFocus("mode")
				}
			}
		}
	}

	m.previewVP.SetContent(m.renderMarkdown(m.editorContent, m.previewVP.Width()))

	return m, cmd
}

func (m Model) updatePreviewView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.previewVP.SetContent(m.renderMarkdown(m.editorContent, m.previewVP.Width()))
	m.previewVP, cmd = m.previewVP.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "e", "p":
			m.currentView = ViewEditor
			if m.editorMode == EditorModePreview {
				m.editorMode = EditorModeSplit
			}
			m.syncEditorLayout()
		}
	}
	return m, cmd
}

func (m Model) updateSearchView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	m.searchInput, cmd = m.searchInput.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			query := m.searchInput.Value()
			if query != "" {
				m.searchQuery = query
				m.currentView = ViewNotes
				return m, m.searchNotesCmd(query)
			}
		case "esc":
			m.searchInput.Reset()
			m.currentView = ViewNotes
			return m, m.loadNotesCmd()
		}
	}

	return m, cmd
}

func (m Model) updateTagsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	// Handle mouse wheel events
	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.tagList.Index() > 0 {
				m.tagList.CursorUp()
			}
		} else {
			if m.tagList.Index() < len(m.tagList.VisibleItems())-1 {
				m.tagList.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) {
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.tagList.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.tagList.VisibleItems()) {
					m.tagList.Select(itemIndex)
				}
			}
		}
		return m, nil
	}

	m.tagList, cmd = m.tagList.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if i, ok := m.tagList.SelectedItem().(TagItem); ok {
				notes, err := m.db.GetNotesByTagName(m.ctx, i.name)
				if err == nil {
					m.notes = notes
					items := make([]list.Item, len(notes))
					for j, n := range notes {
						tags, _ := m.db.GetTagsForNote(m.ctx, n.ID)
						items[j] = NoteItem{note: n, tags: tags}
					}
					m.noteList.SetItems(items)
					m.currentView = ViewNotes
				}
			}
		case "c":
			// Create tag - TODO
		}
	}

	return m, cmd
}

func (m Model) updateDashboardView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "r":
			return m, m.loadStatsCmd()
		}
	}
	return m, nil
}

func (m Model) updateDailyView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			// Load or create daily note
			return m, m.loadDailyNoteCmd()
		}
	}
	return m, nil
}

func (m Model) updateTasksView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.taskList.Index() > 0 {
				m.taskList.CursorUp()
			}
		} else {
			if m.taskList.Index() < len(m.taskList.VisibleItems())-1 {
				m.taskList.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) {
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.taskList.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.taskList.VisibleItems()) {
					m.taskList.Select(itemIndex)
				}
			}
		}
		return m, nil
	}

	m.taskList, cmd = m.taskList.Update(msg)
	return m, cmd
}

func (m Model) updateBacklinksView(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.updateNotesView(msg)
}

func (m Model) updateFoldersView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.folderTree.Index() > 0 {
				m.folderTree.CursorUp()
			}
		} else {
			if m.folderTree.Index() < len(m.folderTree.VisibleItems())-1 {
				m.folderTree.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) {
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.folderTree.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.folderTree.VisibleItems()) {
					m.folderTree.Select(itemIndex)
				}
			}
		}
		return m, nil
	}

	m.folderTree, cmd = m.folderTree.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if i, ok := m.folderTree.SelectedItem().(FolderItem); ok {
				notes, err := m.db.GetNotesByFolder(m.ctx, sql.NullInt64{Int64: i.folder.ID, Valid: true})
				if err == nil {
					m.notes = notes
					items := make([]list.Item, len(notes))
					for j, n := range notes {
						tags, _ := m.db.GetTagsForNote(m.ctx, n.ID)
						items[j] = NoteItem{note: n, tags: tags}
					}
					m.noteList.SetItems(items)
					m.currentView = ViewNotes
				}
			}
		}
	}

	return m, cmd
}

func (m Model) updateHelpView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "?", "esc", "q":
			m.currentView = m.previousView
			m.showHelp = false
		}
	}
	return m, nil
}

func (m Model) updateSettingsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q", ",":
			m.currentView = ViewNotes
		}
	}
	return m, nil
}

func (m Model) updateGraphView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q", "G":
			m.currentView = ViewNotes
		}
	}
	return m, nil
}

func (m Model) updateHistoryView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.versionList.Index() > 0 {
				m.versionList.CursorUp()
			}
		} else {
			if m.versionList.Index() < len(m.versionList.VisibleItems())-1 {
				m.versionList.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) {
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.versionList.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.versionList.VisibleItems()) {
					m.versionList.Select(itemIndex)
				}
			}
		}
		return m, nil
	}

	m.versionList, cmd = m.versionList.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if i, ok := m.versionList.SelectedItem().(VersionItem); ok {
				// Show version content
				m.isCreating = false
				m.openEditor(i.version.Title, i.version.Content, "content")
			}
		case "esc", "H":
			m.currentView = ViewNotes
		}
	}

	return m, cmd
}

func (m Model) updateTemplatesView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.componentsInitialized {
		return m, nil
	}

	switch mouseMsg := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Y < 0 {
			if m.templateList.Index() > 0 {
				m.templateList.CursorUp()
			}
		} else {
			if m.templateList.Index() < len(m.templateList.VisibleItems())-1 {
				m.templateList.CursorDown()
			}
		}
		return m, nil
	case tea.MouseClickMsg:
		mouse := tea.Mouse(mouseMsg)
		if mouse.Button == tea.MouseButton(1) {
			listStartY := 4
			if mouse.Y > listStartY && mouse.Y < listStartY+m.templateList.Height() {
				itemIndex := mouse.Y - listStartY
				if itemIndex >= 0 && itemIndex < len(m.templateList.VisibleItems()) {
					m.templateList.Select(itemIndex)
				}
			}
		}
		return m, nil
	}

	m.templateList, cmd = m.templateList.Update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if i, ok := m.templateList.SelectedItem().(TemplateItem); ok {
				// Create note from template
				m.isCreating = true
				m.editorDirty = false
				m.currentNote = nil
				m.openEditor("", i.template.Content, "title")
			}
		case "c":
			// Create template - TODO
		case "esc", "v":
			m.currentView = ViewNotes
		}
	}

	return m, cmd
}

func (m *Model) saveNoteCmd() tea.Cmd {
	return func() tea.Msg {
		title := m.editorTitle
		content := m.editorContent
		content = strings.TrimSpace(content)

		if title == "" {
			title = "Untitled"
		}

		var note db.Note
		var err error

		if m.isCreating {
			note, err = m.db.CreateNote(m.ctx, db.CreateNoteParams{Title: title, Content: content})
		} else if m.currentNote != nil {
			note, err = m.db.UpdateNote(m.ctx, db.UpdateNoteParams{ID: m.currentNote.ID, Title: title, Content: content})
		} else {
			return fmt.Errorf("no note selected and not creating new note")
		}

		if err != nil {
			return err
		}

		return noteSavedMsg{note: note}
	}
}

func (m *Model) loadDailyNoteCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		today := time.Now().Format("2006-01-02")
		title := "Daily Note " + today

		note, err := m.db.GetNoteByTitle(m.ctx, title)
		if err != nil {
			note, err = m.db.CreateNote(m.ctx, db.CreateNoteParams{Title: title, Content: "# " + today + "\n\n"})
			if err != nil {
				return err
			}
		}

		m.currentNote = &note
		m.isCreating = false
		m.openEditor(note.Title, note.Content, "content")

		return notesLoadedMsg{notes: m.notes}
	}
}

func (m Model) View() tea.View {
	var content string

	if m.quitting {
		content = mutedStyle.Render("Goodbye!")
	} else if m.loading {
		content = modalStyle.Width(m.width - 20).Height(m.height - 10).Render(
			lipgloss.JoinVertical(lipgloss.Center, m.spinner.View(), subheadingStyle.Render("Loading...")),
		)
	} else if m.currentView == ViewHelp {
		content = m.renderHelp()
	} else {
		switch m.currentView {
		case ViewNotes:
			content = m.renderMainLayout()
		case ViewEditor:
			content = m.renderEditorLayout()
		case ViewPreview:
			content = m.renderPreviewLayout()
		case ViewSearch:
			content = m.renderSearchLayout()
		case ViewTags:
			content = m.renderTagsLayout()
		case ViewDashboard:
			content = m.renderDashboardLayout()
		case ViewDaily:
			content = m.renderDailyLayout()
		case ViewTasks:
			content = m.renderTasksLayout()
		case ViewBacklinks:
			content = m.renderBacklinksLayout()
		case ViewFolders:
			content = m.renderFoldersLayout()
		case ViewSettings:
			content = m.renderSettingsLayout()
		case ViewGraph:
			content = m.renderGraphLayout()
		case ViewHistory:
			content = m.renderHistoryLayout()
		case ViewTemplates:
			content = m.renderTemplatesLayout()
		default:
			content = "Unknown view"
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	v.WindowTitle = "Noted"

	return v
}

func (m Model) renderHelp() string {
	help := `
+----------------------------------------------------------------------+
|                           KEYBOARD SHORTCUTS                          |
+----------------------------------------------------------------------+

  NAVIGATION
    j/k or up/down    Navigate up/down
    Enter             Select / Open
    h                 Back / Left
    l                 Forward / Right
    Esc               Back / Cancel
    ?                 Show this help
    q                 Quit

  VIM-STYLE (prefix with g)
    g then h          Go to Notes (first view)
    g then j          Go to next view (Tags)
    g then k          Go to previous view (Settings)
    g then l          Go to Search
    g then g          Go to Notes

  NOTES
    n                 New note
    e                 Edit note
    d                 Delete note
    p                 Pin/unpin note
    /                 Search notes
    v                 Templates

  VIEWS
    t                 Tags
    D                 Dashboard
    l                 Daily notes
    T                 Tasks
    b                 Backlinks
    f                 Folders
    G                 Graph view
    H                 History (when note selected)
    ,                 Settings

  EDITOR (internal)
    Ctrl+S            Save note
    Ctrl+P            Toggle split/edit
    p                 Full preview
    Tab               Cycle focus
    Esc               Back to notes

  SIDEBAR
    Type in sidebar   Live filter (press Enter to search)

+----------------------------------------------------------------------+
`
	centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalStyle.Render(help))
	return centered
}

func (m Model) renderMainLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	content := m.noteList.View()

	footer := m.renderFooter("[n]new [e]edit [d]del [p]pin [/]search [t]tags [D]dash [l]daily [T]tasks [b]links [f]folders [G]graph [?]help [q]quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(sidebarWidth).Render(sidebar),
			lipgloss.NewStyle().Width(m.width-sidebarWidth-1).Render(content),
		),
		footer,
	)
}

func (m Model) renderSidebar() string {
	search := inputStyle.Width(31).Render(m.searchInput.View())

	// Navigation
	navItems := ""
	nav := []string{"Notes", "Tags", "Folders", "Templates", "Dashboard", "Daily", "Tasks", "Settings"}
	views := []ViewName{ViewNotes, ViewTags, ViewFolders, ViewTemplates, ViewDashboard, ViewDaily, ViewTasks, ViewSettings}
	for i, item := range nav {
		prefix := "  "
		if m.currentView == views[i] {
			prefix = "► "
		}
		navItems += prefix + item + "\n"
	}

	stats := fmt.Sprintf("%d notes  |  %d tags", len(m.notes), len(m.tags))

	sidebar := panelStyle.Width(35).Height(m.height - 3).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("noted"),
			search,
			lipgloss.NewStyle().MarginTop(1).Render(navItems),
			mutedStyle.MarginTop(2).Render(stats),
		),
	)

	return sidebar
}

func (m Model) renderFooter(help string) string {
	statusLeft := ""
	if m.currentNote != nil {
		statusLeft = "Editing: " + m.currentNote.Title
	} else if m.searchQuery != "" {
		statusLeft = fmt.Sprintf("Search: %s (%d results)", m.searchQuery, len(m.searchResults))
	} else {
		statusLeft = fmt.Sprintf("%d notes", len(m.notes))
	}

	if m.editorDirty {
		statusLeft += " *"
	}

	helpLen := lipgloss.Width(help)
	availableSpace := m.width - helpLen - 2
	if availableSpace < 0 {
		availableSpace = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		statusStyle.Width(availableSpace).Render(statusLeft),
		statusStyle.Render(help),
	)
}

func (m Model) renderEditorLayout() string {
	metrics := m.editorLayoutMetrics()

	var metaParts []string
	if m.isCreating {
		metaParts = append(metaParts, "new")
	}
	if m.isPinned {
		metaParts = append(metaParts, "pinned")
	}
	if m.currentNote != nil {
		metaParts = append(metaParts, fmt.Sprintf("id:%d", m.currentNote.ID))
	}
	metaLine := strings.Join(metaParts, " | ")

	tabStyle := lipgloss.NewStyle().Foreground(textCol).Background(surfaceAlt)
	tabActive := tabStyle.Foreground(background).Background(primary).Bold(true)
	makeTab := func(label, focus string) string {
		if m.editorFocus == focus {
			return tabActive.Render(label)
		}
		return tabStyle.Render(label)
	}

	tabs := []string{makeTab("[mode]", "mode"), makeTab("[title]", "title"), makeTab("[content]", "content")}
	if m.editorMode == EditorModeSplit {
		tabs = append(tabs, makeTab("[preview]", "preview"))
	}
	headerStyle := lipgloss.NewStyle().Foreground(primary).Bold(true)
	header := headerStyle.Render("Editor") + " " + strings.Join(tabs, " ")

	modeLabel := lipgloss.NewStyle().Foreground(mutedText).Render("Mode:")
	modeFieldView := "< " + string(m.editorMode) + " >"
	modePercent := 0.0
	switch m.editorMode {
	case EditorModeEdit:
		modePercent = 1.0
	case EditorModePreview:
		modePercent = 0.5
	}
	modeSlider := m.modeBar.ViewAs(modePercent)
	modeFieldWidth := max(8, metrics.Mode.W-lipgloss.Width(modeSlider)-1)
	modeRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		modeLabel,
		" ",
		lipgloss.NewStyle().Width(modeFieldWidth).Render(modeFieldView),
		" ",
		modeSlider,
	)

	titleLabel := lipgloss.NewStyle().Foreground(mutedText).Render("Title:")
	titleFieldView := m.editorTitle
	if m.editorTitleField != nil {
		titleFieldView = m.editorTitleField.View()
	}
	titleRow := lipgloss.JoinHorizontal(lipgloss.Left, titleLabel, " ", lipgloss.NewStyle().Width(metrics.Title.W).Render(titleFieldView))

	contentPct := m.contentArea.ScrollPercent()
	previewPct := m.previewVP.ScrollPercent()
	contentLabel := lipgloss.NewStyle().Foreground(mutedText).Render(fmt.Sprintf("Markdown %3.0f%%", contentPct*100))
	previewLabel := lipgloss.NewStyle().Foreground(mutedText).Render(fmt.Sprintf("Preview %3.0f%%", previewPct*100))
	contentBar := m.contentBar.ViewAs(contentPct)
	previewBar := m.previewBar.ViewAs(previewPct)

	contentBorder := lipgloss.NewStyle().
		Width(max(1, metrics.Content.W-2)).
		Height(max(1, metrics.Content.H-2)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border)
	if m.editorFocus == "content" {
		contentBorder = contentBorder.BorderForeground(primary)
	}

	previewBorder := lipgloss.NewStyle().
		Width(max(1, metrics.Preview.W-2)).
		Height(max(1, metrics.Preview.H-2)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border)
	if m.editorFocus == "preview" {
		previewBorder = previewBorder.BorderForeground(primary)
	}

	contentStatus := lipgloss.NewStyle().Width(metrics.Content.W).Render(lipgloss.JoinHorizontal(lipgloss.Left, contentLabel, " ", contentBar))
	contentPane := lipgloss.NewStyle().Width(metrics.Content.W).Render(
		lipgloss.JoinVertical(lipgloss.Left, contentStatus, contentBorder.Render(m.contentArea.View())),
	)
	body := contentPane
	if m.editorMode == EditorModeSplit {
		previewStatus := lipgloss.NewStyle().Width(metrics.Preview.W).Render(lipgloss.JoinHorizontal(lipgloss.Left, previewLabel, " ", previewBar))
		previewPane := lipgloss.NewStyle().Width(metrics.Preview.W).Render(
			lipgloss.JoinVertical(lipgloss.Left, previewStatus, previewBorder.Render(m.previewVP.View())),
		)
		body = lipgloss.JoinHorizontal(lipgloss.Top, contentPane, "  ", previewPane)
	}

	editorTop := []string{header, modeRow, titleRow, lipgloss.NewStyle().Foreground(mutedText).Render(metaLine)}
	editorTop = append(editorTop, body)

	editorPanel := lipgloss.NewStyle().
		Width(metrics.Editor.W).
		Height(metrics.Editor.H).
		Background(surface).
		Render(lipgloss.JoinVertical(lipgloss.Left, editorTop...))

	buttonStyle := lipgloss.NewStyle().
		Foreground(textCol).
		Background(surfaceAlt).
		Padding(0, 1)
	buttonHot := buttonStyle.Background(accent).Foreground(background)
	action := func(label string, active bool) string {
		if active {
			return buttonHot.Width(metrics.Sidebar.W - 2).Render(label)
		}
		return buttonStyle.Width(metrics.Sidebar.W - 2).Render(label)
	}

	sidebarLines := []string{
		lipgloss.NewStyle().Foreground(primary).Bold(true).Render("Actions"),
		action("Ctrl+S Save", false),
		action("Ctrl+P Split/Edit", false),
		action("P Full preview", false),
		action("Tab Focus cycle", false),
		action("Esc Back", m.editorBackArmed),
		"",
		lipgloss.NewStyle().Foreground(mutedText).Render("Mouse: click to focus/cursor"),
	}

	sidebarPanel := lipgloss.NewStyle().
		Width(metrics.Sidebar.W).
		Height(metrics.Sidebar.H).
		Background(surfaceAlt).
		Render(lipgloss.JoinVertical(lipgloss.Left, sidebarLines...))

	footer := m.renderFooter("[Ctrl+S]save [Ctrl+P]split/edit [P]preview [Tab]focus [Esc]back")

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, editorPanel)
	main = lipgloss.NewStyle().MaxHeight(max(1, m.height-1)).Render(main)

	return lipgloss.JoinVertical(lipgloss.Left, main, footer)
}

func (m Model) renderPreviewLayout() string {
	preview := m.renderMarkdown(m.editorContent, m.previewVP.Width())
	m.previewVP.SetContent(preview)

	sidebarWidth := 35
	sidebar := panelStyle.Width(sidebarWidth).Height(m.height - 6).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Preview"),
			mutedStyle.Render("Rendered from current note"),
			lipgloss.NewStyle().MarginTop(2).Render("Press p or Esc to return"),
		),
	)

	contentPanel := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		m.previewVP.View(),
	)

	footer := m.renderFooter("[p]edit [Esc]back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, contentPanel),
		footer,
	)
}

func (m Model) renderSearchLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	input := inputFocusedStyle.Width(60).Render(m.searchInput.View())
	searchHeader := headingStyle.Render("Search") + "\n" + input + "\n\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, searchHeader, m.noteList.View()),
	)

	footer := m.renderFooter("Enter:search Esc:cancel")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderTagsLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Tags") + "\n" +
		mutedStyle.Render("Enter to filter notes") + "\n\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, m.tagList.View()),
	)

	footer := m.renderFooter("Enter:filter Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderDashboardLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	card := func(title, value, emoji string) string {
		return cardStyle.Width(22).Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().Bold(true).Foreground(primary).Render(emoji),
				headingStyle.Render(value),
				mutedStyle.Render(title),
			),
		)
	}

	cards := lipgloss.JoinHorizontal(
		lipgloss.Top,
		card("Total Notes", strconv.FormatInt(m.stats.TotalNotes, 10), "📝"),
		card("Tags", strconv.FormatInt(m.stats.TotalTags, 10), "🏷️"),
		card("Pinned", strconv.FormatInt(m.stats.PinnedNotes, 10), "📌"),
		card("Expired", strconv.FormatInt(m.stats.ExpiredNotes, 10), "⏰"),
	)

	header := headingStyle.Render("Dashboard")

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, cards),
	)

	footer := m.renderFooter("r:refresh Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderDailyLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	noteTitle := "Daily Note " + time.Now().Format("2006-01-02")

	// If we already have the daily note loaded, show editor
	if m.currentNote != nil && strings.HasPrefix(m.currentNote.Title, "Daily Note ") {
		return m.renderEditorLayout()
	}

	// Show loading/creation prompt
	header := headingStyle.Render("Daily Note: "+noteTitle) + "\n" +
		mutedStyle.Render("Press Enter to create today's note") + "\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header),
	)

	footer := m.renderFooter("Enter:create Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderTasksLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Tasks") + "\n" +
		mutedStyle.Render("From all notes") + "\n\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, m.taskList.View()),
	)

	footer := m.renderFooter("Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderBacklinksLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	var header string
	if m.currentNote != nil {
		header = headingStyle.Render("Backlinks: " + m.currentNote.Title)
	} else {
		header = headingStyle.Render("Backlinks")
	}
	header += "\n" + mutedStyle.Render("Notes linking here") + "\n\n"

	var content string
	if len(m.backlinks) == 0 {
		content = panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 15).Render(mutedStyle.Render("No backlinks"))
	} else {
		items := make([]list.Item, len(m.backlinks))
		for i, n := range m.backlinks {
			tags, _ := m.db.GetTagsForNote(m.ctx, n.ID)
			items[i] = NoteItem{note: n, tags: tags}
		}
		m.noteList.SetItems(items)
		content = panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
			lipgloss.JoinVertical(lipgloss.Left, header, m.noteList.View()),
		)
	}

	footer := m.renderFooter("Enter:open Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderFoldersLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Folders") + "\n" +
		mutedStyle.Render("Press Enter to view") + "\n\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, m.folderTree.View()),
	)

	footer := m.renderFooter("Enter:open Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderSettingsLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Settings")

	settings := `
Editor:     Internal textarea
Database:   ~/.local/share/noted/noted.db
Theme:      Nord (terminal colors)

TUI Features:
• Mouse support enabled
• Vim keybindings
• Markdown preview (Glamour)
• Graph view (Harmonica)

Press Esc to go back
`

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.NewStyle().MarginTop(1).Render(settings)),
	)

	footer := m.renderFooter("Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderGraphLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Graph View")

	// Get all notes and links for graph
	notes, _ := m.db.GetAllNotes(m.ctx)
	links, _ := m.db.GetAllNoteLinks(m.ctx)

	// Calculate simple force-directed layout
	graphViz := renderGraphASCII(notes, links, m.width-sidebarWidth-10, m.height-15)

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.NewStyle().MarginTop(1).Render(graphViz)),
	)

	footer := m.renderFooter("Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

// GraphNode represents a node in the graph for visualization
type GraphNode struct {
	ID     int64
	Title  string
	X, Y   float64
	Vx, Vy float64
}

// renderGraphASCII creates a simple ASCII force-directed graph visualization
func renderGraphASCII(notes []db.Note, links []db.NoteLink, width, height int) string {
	if len(notes) == 0 {
		return "No notes to display"
	}

	// Create nodes map
	nodes := make(map[int64]*GraphNode)
	nodeList := make([]*GraphNode, len(notes))

	for i, note := range notes {
		// Position nodes in a circle initially
		angle := 2 * math.Pi * float64(i) / float64(len(notes))
		radius := math.Min(float64(width), float64(height)) / 3
		node := &GraphNode{
			ID:    note.ID,
			Title: truncateTitle(note.Title, 12),
			X:     float64(width/2) + radius*math.Cos(angle),
			Y:     float64(height/2) + radius*math.Sin(angle),
		}
		nodes[note.ID] = node
		nodeList[i] = node
	}

	// Run force-directed simulation using harmonica springs
	spring := harmonica.NewSpring(harmonica.FPS(30), 4.0, 0.5)

	// Create edges for spring connections
	edges := make([][2]int64, 0)
	linkMap := make(map[int64][]int64)
	for _, link := range links {
		if _, ok := nodes[link.SourceNoteID]; ok {
			if _, ok := nodes[link.TargetNoteID]; ok {
				edges = append(edges, [2]int64{link.SourceNoteID, link.TargetNoteID})
				linkMap[link.SourceNoteID] = append(linkMap[link.SourceNoteID], link.TargetNoteID)
			}
		}
	}

	// Run simulation iterations
	for iter := 0; iter < 50; iter++ {
		for _, node := range nodeList {
			node.Vx = 0
			node.Vy = 0
		}

		// Repulsion between all nodes
		for i := 0; i < len(nodeList); i++ {
			for j := i + 1; j < len(nodeList); j++ {
				n1 := nodeList[i]
				n2 := nodeList[j]

				dx := n2.X - n1.X
				dy := n2.Y - n1.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < 1 {
					dist = 1
				}

				// Repulsion force
				force := 1000 / (dist * dist)
				fx := (dx / dist) * force
				fy := (dy / dist) * force

				n1.Vx -= fx
				n1.Vy -= fy
				n2.Vx += fx
				n2.Vy += fy
			}
		}

		// Attraction along edges
		for _, edge := range edges {
			n1 := nodes[edge[0]]
			n2 := nodes[edge[1]]
			if n1 == nil || n2 == nil {
				continue
			}

			dx := n2.X - n1.X
			dy := n2.Y - n1.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1 {
				dist = 1
			}

			// Attraction force
			force := dist * 0.1
			fx := (dx / dist) * force
			fy := (dy / dist) * force

			n1.Vx += fx
			n1.Vy += fy
			n2.Vx -= fx
			n2.Vy -= fy
		}

		// Center gravity
		cx := float64(width / 2)
		cy := float64(height / 2)
		for _, node := range nodeList {
			node.Vx += (cx - node.X) * 0.01
			node.Vy += (cy - node.Y) * 0.01
		}

		// Apply velocities using harmonica spring
		for _, node := range nodeList {
			vx, vy := spring.Update(node.Vx, node.Vy, 0)
			node.X += vx
			node.Y += vy

			// Keep within bounds
			node.X = math.Max(2, math.Min(float64(width-2), node.X))
			node.Y = math.Max(1, math.Min(float64(height-2), node.Y))
		}
	}

	// Render to ASCII grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Draw edges
	for _, edge := range edges {
		n1 := nodes[edge[0]]
		n2 := nodes[edge[1]]
		if n1 == nil || n2 == nil {
			continue
		}

		// Bresenham's line algorithm
		x0, y0 := int(n1.X), int(n1.Y)
		x1, y1 := int(n2.X), int(n2.Y)

		dx := x1 - x0
		if dx < 0 {
			dx = -dx
		}
		dy := y1 - y0
		if dy < 0 {
			dy = -dy
		}
		dy = -dy
		sx := 1
		if x0 >= x1 {
			sx = -1
		}
		sy := 1
		if y0 >= y1 {
			sy = -1
		}
		err := dx + dy

		for {
			if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
				if grid[y0][x0] == ' ' {
					grid[y0][x0] = '─'
				}
			}
			if x0 == x1 && y0 == y1 {
				break
			}
			e2 := 2 * err
			if e2 >= dy {
				err += dy
				x0 += sx
			}
			if e2 <= dx {
				err += dx
				y0 += sy
			}
		}
	}

	// Draw nodes
	for _, node := range nodeList {
		x, y := int(node.X), int(node.Y)
		if x >= 1 && x < width-1 && y >= 0 && y < height {
			// Draw node as [title]
			title := node.Title
			if len(title) > 10 {
				title = title[:10]
			}
			startX := x - len(title)/2 - 1
			if startX < 1 {
				startX = 1
			}
			if startX+len(title)+2 >= width {
				startX = width - len(title) - 2
			}

			// Draw [ around node
			if startX > 0 {
				grid[y][startX] = '['
			}
			// Draw title
			for i, c := range title {
				if startX+1+i < width-1 {
					grid[y][startX+1+i] = c
				}
			}
			// Draw ] after title
			if startX+len(title)+1 < width-1 {
				grid[y][startX+len(title)+1] = ']'
			}
		}
	}

	// Convert grid to string
	var sb strings.Builder
	fmt.Fprintf(&sb, "Nodes: %d | Edges: %d\n\n", len(notes), len(links))
	for _, row := range grid {
		sb.WriteString(string(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-2] + ".."
}

func (m Model) renderHistoryLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Version History") + "\n"

	if m.currentNote != nil {
		header += fmt.Sprintf("Note: %s\n\n", m.currentNote.Title)
	}

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, m.versionList.View()),
	)

	footer := m.renderFooter("Enter:view Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}

func (m Model) renderTemplatesLayout() string {
	sidebarWidth := 35
	sidebar := m.renderSidebar()

	header := headingStyle.Render("Templates") + "\n" +
		mutedStyle.Render("Enter to create note") + "\n\n"

	content := panelStyle.Width(m.width - sidebarWidth - 1).Height(m.height - 6).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, m.templateList.View()),
	)

	footer := m.renderFooter("Enter:create Esc:back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content),
		footer,
	)
}
