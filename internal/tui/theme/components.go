package theme

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// ListChrome applies Nord styling to a list.Model's chrome (title bar, status bar, pagination,
// filter input). It does not touch the row delegate — use NewItemDelegate for that.
func ListChrome(l *list.Model) {
	s := list.DefaultStyles(true)
	s.Title = s.Title.Foreground(Bg).Background(Primary).Bold(true).Padding(0, 1)
	s.TitleBar = s.TitleBar.Padding(0, 0, 1, 2)
	s.StatusBar = s.StatusBar.Foreground(Muted)
	s.StatusEmpty = s.StatusEmpty.Foreground(Border)
	s.StatusBarActiveFilter = s.StatusBarActiveFilter.Foreground(Text)
	s.StatusBarFilterCount = s.StatusBarFilterCount.Foreground(Border)
	s.NoItems = s.NoItems.Foreground(Border)
	s.PaginationStyle = s.PaginationStyle.PaddingLeft(2)
	s.ActivePaginationDot = s.ActivePaginationDot.Foreground(Primary)
	s.InactivePaginationDot = s.InactivePaginationDot.Foreground(Border)
	s.DividerDot = s.DividerDot.Foreground(Border)
	s.Filter.Cursor.Color = Primary
	s.Filter.Focused.Prompt = s.Filter.Focused.Prompt.Foreground(Primary)
	s.Filter.Focused.Text = s.Filter.Focused.Text.Foreground(Text)
	l.Styles = s
}

// NewItemDelegate returns a Nord-themed list delegate. Height/Spacing are set explicitly so mouse
// row math is predictable (stride = delegate.Height() + delegate.Spacing()). Callers that need
// per-row click zones can embed the returned delegate and override Render with a bubblezone Mark.
func NewItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	is := list.NewDefaultItemStyles(true)
	is.NormalTitle = is.NormalTitle.Foreground(Text)
	is.NormalDesc = is.NormalDesc.Foreground(Muted)
	is.SelectedTitle = is.SelectedTitle.Foreground(Primary).BorderForeground(Primary).Bold(true)
	is.SelectedDesc = is.SelectedDesc.Foreground(Secondary).BorderForeground(Primary)
	is.DimmedTitle = is.DimmedTitle.Foreground(Muted)
	is.DimmedDesc = is.DimmedDesc.Foreground(Border)
	is.FilterMatch = is.FilterMatch.Foreground(Warning).Underline(true)
	d.Styles = is
	d.ShowDescription = true
	d.SetHeight(2)
	d.SetSpacing(1)
	return d
}

// List applies both Nord chrome and the default Nord row delegate to a list.Model.
func List(l *list.Model) {
	ListChrome(l)
	l.SetDelegate(NewItemDelegate())
}

// TextInput applies Nord styling to a textinput.Model.
func TextInput(ti *textinput.Model) {
	s := ti.Styles()
	s.Focused.Text = s.Focused.Text.Foreground(Text)
	s.Focused.Placeholder = s.Focused.Placeholder.Foreground(Border)
	s.Focused.Prompt = s.Focused.Prompt.Foreground(Primary)
	s.Focused.Suggestion = s.Focused.Suggestion.Foreground(Border)
	s.Blurred.Text = s.Blurred.Text.Foreground(Muted)
	s.Blurred.Placeholder = s.Blurred.Placeholder.Foreground(Border)
	s.Blurred.Prompt = s.Blurred.Prompt.Foreground(Accent)
	s.Cursor.Color = Primary
	s.Cursor.Shape = tea.CursorBar
	s.Cursor.Blink = true
	ti.SetStyles(s)
}

// Textarea applies Nord styling to a textarea.Model.
func Textarea(ta *textarea.Model) {
	s := ta.Styles()
	s.Focused.Base = s.Focused.Base.Background(Bg)
	s.Focused.Text = s.Focused.Text.Foreground(Text)
	s.Focused.Placeholder = s.Focused.Placeholder.Foreground(Border)
	s.Focused.Prompt = s.Focused.Prompt.Foreground(Primary)
	s.Focused.LineNumber = s.Focused.LineNumber.Foreground(Border)
	s.Focused.CursorLine = s.Focused.CursorLine.Background(Surface)
	s.Focused.CursorLineNumber = s.Focused.CursorLineNumber.Foreground(Primary)
	s.Focused.EndOfBuffer = s.Focused.EndOfBuffer.Foreground(SurfaceHi)
	s.Blurred.Base = s.Blurred.Base.Background(Bg)
	s.Blurred.Text = s.Blurred.Text.Foreground(Muted)
	s.Blurred.LineNumber = s.Blurred.LineNumber.Foreground(Border)
	s.Blurred.Prompt = s.Blurred.Prompt.Foreground(Border)
	s.Cursor.Color = Primary
	s.Cursor.Shape = tea.CursorBlock
	s.Cursor.Blink = true
	ta.SetStyles(s)
}

// Spinner applies Nord styling to a spinner.Model.
func Spinner(sp *spinner.Model) {
	sp.Style = lipgloss.NewStyle().Foreground(Primary)
}

// Progress applies Nord styling to a progress.Model (set after construction).
func Progress(p *progress.Model) {
	p.FullColor = Primary
	p.EmptyColor = SurfaceHi
	p.PercentageStyle = lipgloss.NewStyle().Foreground(Muted)
}

// Viewport applies Nord styling to a viewport.Model.
func Viewport(vp *viewport.Model) {
	vp.Style = lipgloss.NewStyle().Background(Bg).Foreground(Text)
	vp.HighlightStyle = lipgloss.NewStyle().Background(Border).Foreground(Text)
	vp.SelectedHighlightStyle = lipgloss.NewStyle().Background(Primary).Foreground(Bg)
}

// Help applies Nord styling to a help.Model.
func Help(h *help.Model) {
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(Primary)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(Muted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(Border)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(Border)
}

// HuhTheme returns a Nord huh.Theme suitable for form.WithTheme(...).
func HuhTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)

		f := &t.Focused
		f.Base = f.Base.BorderForeground(Primary)
		f.Card = f.Base
		f.Title = f.Title.Foreground(Primary).Bold(true)
		f.NoteTitle = f.NoteTitle.Foreground(Primary).Bold(true).MarginBottom(1)
		f.Description = f.Description.Foreground(Border)
		f.Directory = f.Directory.Foreground(Secondary)
		f.File = f.File.Foreground(Text)
		f.ErrorIndicator = f.ErrorIndicator.Foreground(ErrorC)
		f.ErrorMessage = f.ErrorMessage.Foreground(ErrorC)
		f.SelectSelector = f.SelectSelector.Foreground(Warning)
		f.NextIndicator = f.NextIndicator.Foreground(Warning)
		f.PrevIndicator = f.PrevIndicator.Foreground(Warning)
		f.Option = f.Option.Foreground(Text)
		f.MultiSelectSelector = f.MultiSelectSelector.Foreground(Warning)
		f.SelectedOption = f.SelectedOption.Foreground(Success)
		f.SelectedPrefix = lipgloss.NewStyle().Foreground(Success).SetString("✓ ")
		f.UnselectedPrefix = lipgloss.NewStyle().Foreground(Border).SetString("• ")
		f.UnselectedOption = f.UnselectedOption.Foreground(Muted)
		f.FocusedButton = f.FocusedButton.Foreground(Bg).Background(Primary)
		f.Next = f.FocusedButton
		f.BlurredButton = f.BlurredButton.Foreground(Muted).Background(SurfaceHi)

		f.TextInput.Cursor = f.TextInput.Cursor.Foreground(Primary)
		f.TextInput.Text = f.TextInput.Text.Foreground(Text)
		f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(Border)
		f.TextInput.Prompt = f.TextInput.Prompt.Foreground(Primary)

		t.Blurred = t.Focused
		t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
		t.Blurred.Card = t.Blurred.Base
		t.Blurred.NextIndicator = lipgloss.NewStyle()
		t.Blurred.PrevIndicator = lipgloss.NewStyle()

		t.Group.Title = t.Focused.Title
		t.Group.Description = t.Focused.Description
		t.Help.ShortKey = lipgloss.NewStyle().Foreground(Primary)
		t.Help.ShortDesc = lipgloss.NewStyle().Foreground(Muted)
		t.Help.FullKey = t.Help.ShortKey
		t.Help.FullDesc = t.Help.ShortDesc

		return t
	})
}
