package tui

import (
	"database/sql"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

var nowFunc = time.Now

type MouseHandler struct {
	width       int
	height      int
	hoverX      int
	hoverY      int
	lastClick   time.Time
	doubleClick bool
}

const (
	sidebarWidth          = 35
	sidebarPaddingTop     = 1
	sidebarTitleHeight    = 2
	sidebarSearchHeight   = 3
	sidebarNavMarginTop   = 1
	sidebarNavItemCount   = 8
	mainListStartY        = 4
	defaultListItemStride = 3
)

func NewMouseHandler() *MouseHandler {
	return &MouseHandler{}
}

func (mh *MouseHandler) SetSize(width, height int) {
	mh.width = width
	mh.height = height
}

func (mh *MouseHandler) UpdateHover(x, y int) {
	mh.hoverX = x
	mh.hoverY = y
}

func (mh *MouseHandler) GetHoverZone(currentView ViewName) string {
	return mh.detectZone(mh.hoverX, mh.hoverY, currentView)
}

func (mh *MouseHandler) IsHovering(x, y int) bool {
	return mh.hoverX == x && mh.hoverY == y
}

func (mh *MouseHandler) Handle(msg tea.MouseMsg, model *Model) (tea.Model, tea.Cmd) {
	var mouse tea.Mouse

	switch e := msg.(type) {
	case tea.MouseMotionMsg:
		m := e.Mouse()
		mh.UpdateHover(m.X, m.Y)
		return model, nil
	case tea.MouseWheelMsg:
		mouse = e.Mouse()
		mh.doubleClick = false
	case tea.MouseClickMsg:
		mouse = e.Mouse()
		if mouse.Button == tea.MouseLeft {
			now := nowFunc()
			mh.doubleClick = now.Sub(mh.lastClick) < 300*time.Millisecond
			mh.lastClick = now
		} else {
			mh.doubleClick = false
		}
	case tea.MouseReleaseMsg:
		return model, nil
	default:
		return model, nil
	}

	zone := mh.detectZone(mouse.X, mouse.Y, model.currentView)
	if model.currentView == ViewEditor {
		zone = mh.detectEditorZone(mouse.X, mouse.Y, model)
	}

	switch zone {
	case "sidebar_nav":
		return mh.handleSidebarNav(mouse, model)
	case "sidebar_search":
		return mh.handleSidebarSearch(model)
	case "main_list":
		return mh.handleMainList(mouse, model)
	case "editor_content":
		return mh.handleEditorContent(mouse, model)
	case "editor_preview":
		return mh.handleEditorPreview(mouse, model)
	case "editor_mode":
		return mh.handleEditorMode(mouse, model)
	case "editor_title":
		return mh.handleEditorTitle(mouse, model)
	case "editor_action":
		return mh.handleEditorAction(mouse, model)
	}

	return model, nil
}

func (mh *MouseHandler) detectZone(x, y int, currentView ViewName) string {
	searchStartY, searchEndY := mh.sidebarSearchRange()
	navStartY := mh.sidebarNavStartY()

	if currentView == ViewEditor {
		return "editor_other"
	}

	// Other views
	if x < sidebarWidth {
		if y >= searchStartY && y <= searchEndY {
			return "sidebar_search"
		}
		if y >= navStartY && y < navStartY+sidebarNavItemCount {
			return "sidebar_nav"
		}
		return "sidebar_other"
	}

	if y >= 4 && y < mh.height-2 {
		return "main_list"
	}

	return "other"
}

func (mh *MouseHandler) detectEditorZone(x, y int, model *Model) string {
	metrics := model.editorLayoutMetrics()

	if action, ok := metrics.ActionAt(x, y); ok && action != "" {
		return "editor_action"
	}
	if metrics.Mode.Contains(x, y) {
		return "editor_mode"
	}
	if metrics.Title.Contains(x, y) {
		return "editor_title"
	}
	if metrics.Content.Contains(x, y) {
		return "editor_content"
	}
	if model.editorMode == EditorModeSplit && metrics.Preview.Contains(x, y) {
		return "editor_preview"
	}
	if metrics.Sidebar.Contains(x, y) {
		return "sidebar_other"
	}
	return "editor_other"
}

func (mh *MouseHandler) sidebarSearchRange() (int, int) {
	start := sidebarPaddingTop + sidebarTitleHeight
	end := start + sidebarSearchHeight - 1
	return start, end
}

func (mh *MouseHandler) sidebarNavStartY() int {
	_, searchEnd := mh.sidebarSearchRange()
	return searchEnd + 1 + sidebarNavMarginTop
}

func (mh *MouseHandler) handleSidebarNav(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	navItems := []ViewName{
		ViewNotes,
		ViewTags,
		ViewFolders,
		ViewTemplates,
		ViewDashboard,
		ViewDaily,
		ViewTasks,
		ViewSettings,
	}

	navIndex := mouse.Y - mh.sidebarNavStartY()
	if navIndex >= 0 && navIndex < len(navItems) {
		view := navItems[navIndex]
		switch view {
		case ViewDashboard:
			model.currentView = view
			return model, model.loadStatsCmd()
		case ViewDaily:
			model.currentView = view
			return model, model.loadDailyNoteCmd()
		case ViewTasks:
			model.currentView = view
			return model, model.loadTasksCmd()
		default:
			model.currentView = view
			return model, nil
		}
	}

	return model, nil
}

func (mh *MouseHandler) handleSidebarSearch(model *Model) (tea.Model, tea.Cmd) {
	model.searchInput.Focus()
	model.currentView = ViewSearch
	return model, nil
}

func (mh *MouseHandler) handleEditorTitle(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseButton(1) {
		model.setEditorFocus("title")
	}
	return model, nil
}

func (mh *MouseHandler) handleEditorMode(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseButton(1) {
		model.setEditorFocus("mode")
	}
	return model, nil
}

func (mh *MouseHandler) handleEditorContent(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	// Handle mouse wheel for scrolling
	if mouse.Button == tea.MouseButton(4) {
		for i := 0; i < 3; i++ {
			model.contentArea.CursorUp()
		}
		return model, nil
	}
	if mouse.Button == tea.MouseButton(5) {
		for i := 0; i < 3; i++ {
			model.contentArea.CursorDown()
		}
		model.editorContent = model.contentArea.Value()
		return model, nil
	}

	// Handle left click - focus content area
	if mouse.Button == tea.MouseButton(1) {
		model.setEditorFocus("content")
		mh.positionContentCursor(mouse.X, mouse.Y, model)

		return model, nil
	}

	return model, nil
}

func (mh *MouseHandler) handleEditorPreview(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseButton(1) {
		model.setEditorFocus("preview")
		return model, nil
	}

	if mouse.Button == tea.MouseButton(4) {
		model.previewVP.ScrollUp(3)
		return model, nil
	}

	if mouse.Button == tea.MouseButton(5) {
		model.previewVP.ScrollDown(3)
		return model, nil
	}

	return model, nil
}

func (mh *MouseHandler) handleEditorAction(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseButton(1) {
		return model, nil
	}

	action, ok := model.editorLayoutMetrics().ActionAt(mouse.X, mouse.Y)
	if !ok {
		return model, nil
	}

	switch action {
	case "focus_mode":
		model.setEditorFocus("mode")
		return model, nil
	case "focus_title":
		model.setEditorFocus("title")
		return model, nil
	case "focus_content":
		model.setEditorFocus("content")
		return model, nil
	case "focus_preview":
		if model.editorMode == EditorModeSplit {
			model.setEditorFocus("preview")
		}
		return model, nil
	case "save":
		model.editorBackArmed = false
		return model, model.saveNoteCmd()
	case "toggle_mode":
		if model.editorMode == EditorModeSplit {
			model.editorMode = EditorModeEdit
			if model.editorFocus == "preview" {
				model.setEditorFocus("content")
			}
		} else {
			model.editorMode = EditorModeSplit
		}
		model.syncEditorLayout()
		model.previewVP.SetContent(model.renderMarkdown(model.editorContent, model.previewVP.Width()))
		return model, nil
	case "full_preview":
		model.editorMode = EditorModePreview
		model.currentView = ViewPreview
		model.syncEditorLayout()
		return model, nil
	case "focus":
		if model.editorMode == EditorModeSplit {
			switch model.editorFocus {
			case "mode":
				model.setEditorFocus("title")
			case "title":
				model.setEditorFocus("content")
			case "content":
				model.setEditorFocus("preview")
			default:
				model.setEditorFocus("mode")
			}
		} else {
			switch model.editorFocus {
			case "mode":
				model.setEditorFocus("title")
			case "title":
				model.setEditorFocus("content")
			default:
				model.setEditorFocus("mode")
			}
		}
		return model, nil
	case "back":
		if model.editorDirty && !model.editorBackArmed {
			model.editorBackArmed = true
			return model, func() tea.Msg { return toastMsg{message: "Unsaved changes. Click Back again to discard."} }
		}
		model.currentView = ViewNotes
		model.currentNote = nil
		model.editorBackArmed = false
		return model, nil
	}

	return model, nil
}

func (mh *MouseHandler) positionContentCursor(clickX, clickY int, model *Model) {
	metrics := model.editorLayoutMetrics()
	if !metrics.Content.Contains(clickX, clickY) {
		return
	}

	const contentInsetX = 1
	const contentInsetY = 2

	innerX := clickX - metrics.Content.X - contentInsetX
	innerY := clickY - metrics.Content.Y - contentInsetY
	if innerX < 0 {
		innerX = 0
	}
	if innerY < 0 {
		innerY = 0
	}

	wrapWidth := max(1, model.contentArea.Width())
	scroll := model.contentArea.ScrollYOffset()
	targetVisualRow := scroll + innerY

	lines := strings.Split(model.editorContent, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	targetLine := 0
	wrappedLineOffset := 0
	accumVisual := 0
	for i, line := range lines {
		lineWidth := max(1, lipgloss.Width(line))
		rows := (lineWidth + wrapWidth - 1) / wrapWidth
		if line == "" {
			rows = 1
		}
		if targetVisualRow < accumVisual+rows {
			targetLine = i
			wrappedLineOffset = targetVisualRow - accumVisual
			break
		}
		accumVisual += rows
		targetLine = i
		wrappedLineOffset = rows - 1
	}

	lineRunes := []rune(lines[targetLine])
	targetCol := wrappedLineOffset*wrapWidth + innerX
	if targetCol < 0 {
		targetCol = 0
	}
	if targetCol > len(lineRunes) {
		targetCol = len(lineRunes)
	}

	currentLine := model.contentArea.Line()
	for currentLine < targetLine {
		model.contentArea.CursorDown()
		currentLine = model.contentArea.Line()
	}
	for currentLine > targetLine {
		model.contentArea.CursorUp()
		currentLine = model.contentArea.Line()
	}
	model.contentArea.SetCursorColumn(targetCol)
}

func (mh *MouseHandler) handleMainList(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseButton(4) { // Wheel up
		return mh.handleScroll(model, -1)
	}
	if mouse.Button == tea.MouseButton(5) { // Wheel down
		return mh.handleScroll(model, 1)
	}

	if mouse.Button == tea.MouseButton(1) { // Left click
		return mh.handleItemClick(mouse, model)
	}

	return model, nil
}

func (mh *MouseHandler) handleScroll(model *Model, direction int) (tea.Model, tea.Cmd) {
	var currentList *list.Model
	switch model.currentView {
	case ViewNotes:
		currentList = &model.noteList
	case ViewTags:
		currentList = &model.tagList
	case ViewFolders:
		currentList = &model.folderTree
	case ViewTasks:
		currentList = &model.taskList
	case ViewHistory:
		currentList = &model.versionList
	case ViewTemplates:
		currentList = &model.templateList
	default:
		return model, nil
	}

	if direction < 0 {
		currentList.CursorUp()
	} else {
		currentList.CursorDown()
	}

	return model, nil
}

func (mh *MouseHandler) handleItemClick(mouse tea.Mouse, model *Model) (tea.Model, tea.Cmd) {
	listStartY := mainListStartY
	listHeight := model.height - 6

	if mouse.Y < listStartY || mouse.Y >= listStartY+listHeight {
		return model, nil
	}

	row := mouse.Y - listStartY
	if row < 0 {
		return model, nil
	}

	itemOffset := row / defaultListItemStride

	switch model.currentView {
	case ViewNotes:
		start, _ := model.noteList.Paginator.GetSliceBounds(len(model.noteList.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.noteList.VisibleItems()) {
			model.noteList.Select(itemIndex)
			if mh.doubleClick {
				return mh.activateNote(model)
			}
		}
	case ViewTags:
		start, _ := model.tagList.Paginator.GetSliceBounds(len(model.tagList.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.tagList.VisibleItems()) {
			model.tagList.Select(itemIndex)
			return mh.activateTag(model)
		}
	case ViewFolders:
		start, _ := model.folderTree.Paginator.GetSliceBounds(len(model.folderTree.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.folderTree.VisibleItems()) {
			model.folderTree.Select(itemIndex)
			return mh.activateFolder(model)
		}
	case ViewTasks:
		start, _ := model.taskList.Paginator.GetSliceBounds(len(model.taskList.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.taskList.VisibleItems()) {
			model.taskList.Select(itemIndex)
		}
	case ViewHistory:
		start, _ := model.versionList.Paginator.GetSliceBounds(len(model.versionList.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.versionList.VisibleItems()) {
			model.versionList.Select(itemIndex)
			return mh.activateVersion(model)
		}
	case ViewTemplates:
		start, _ := model.templateList.Paginator.GetSliceBounds(len(model.templateList.VisibleItems()))
		itemIndex := start + itemOffset
		if itemIndex >= 0 && itemIndex < len(model.templateList.VisibleItems()) {
			model.templateList.Select(itemIndex)
			return mh.activateTemplate(model)
		}
	}

	return model, nil
}

func (mh *MouseHandler) activateNote(model *Model) (tea.Model, tea.Cmd) {
	if i, ok := model.noteList.SelectedItem().(NoteItem); ok {
		note, err := model.db.GetNote(model.ctx, i.note.ID)
		if err == nil {
			model.currentNote = &note
			model.isCreating = false
			model.isPinned = note.Pinned.Valid && note.Pinned.Bool
			model.openEditor(note.Title, note.Content, "content")
		}
	}
	return model, nil
}

func (mh *MouseHandler) activateTag(model *Model) (tea.Model, tea.Cmd) {
	if i, ok := model.tagList.SelectedItem().(TagItem); ok {
		notes, err := model.db.GetNotesByTagName(model.ctx, i.name)
		if err == nil {
			model.notes = notes
			items := make([]list.Item, len(notes))
			for j, n := range notes {
				tags, _ := model.db.GetTagsForNote(model.ctx, n.ID)
				items[j] = NoteItem{note: n, tags: tags}
			}
			model.noteList.SetItems(items)
			model.currentView = ViewNotes
		}
	}
	return model, nil
}

func (mh *MouseHandler) activateFolder(model *Model) (tea.Model, tea.Cmd) {
	if i, ok := model.folderTree.SelectedItem().(FolderItem); ok {
		notes, err := model.db.GetNotesByFolder(model.ctx, sql.NullInt64{Int64: i.folder.ID, Valid: true})
		if err == nil {
			model.notes = notes
			items := make([]list.Item, len(notes))
			for j, n := range notes {
				tags, _ := model.db.GetTagsForNote(model.ctx, n.ID)
				items[j] = NoteItem{note: n, tags: tags}
			}
			model.noteList.SetItems(items)
			model.currentView = ViewNotes
		}
	}
	return model, nil
}

func (mh *MouseHandler) activateVersion(model *Model) (tea.Model, tea.Cmd) {
	if i, ok := model.versionList.SelectedItem().(VersionItem); ok {
		model.isCreating = false
		model.openEditor(i.version.Title, i.version.Content, "content")
	}
	return model, nil
}

func (mh *MouseHandler) activateTemplate(model *Model) (tea.Model, tea.Cmd) {
	if i, ok := model.templateList.SelectedItem().(TemplateItem); ok {
		model.isCreating = true
		model.editorDirty = false
		model.currentNote = nil
		model.openEditor("", i.template.Content, "title")
	}
	return model, nil
}

func init() {
	_ = (*db.Queries)(nil)
}
