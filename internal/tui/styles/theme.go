package styles

import (
	"charm.land/lipgloss/v2"
)

var (
	Primary   = lipgloss.Color("#88C0D0")
	Secondary = lipgloss.Color("#81A1C1")
	Accent    = lipgloss.Color("#5E81AC")

	Success = lipgloss.Color("#A3BE8C")
	Warning = lipgloss.Color("#EBCB8B")
	Error   = lipgloss.Color("#BF616A")
	Info    = lipgloss.Color("#88C0D0")

	Background = lipgloss.Color("#2E3440")
	Surface    = lipgloss.Color("#3B4252")
	SurfaceAlt = lipgloss.Color("#434C5E")
	Border     = lipgloss.Color("#4C566A")

	Text      = lipgloss.Color("#ECEFF4")
	MutedText = lipgloss.Color("#D8DEE9")

	White = lipgloss.Color("#FFFFFF")
)

var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Background)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(Border)

	RoundedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1)
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	HeadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Text).
			MarginBottom(1)

	SubheadingStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			MarginBottom(1)

	BodyStyle = lipgloss.NewStyle().
			Foreground(Text)

	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedText)
)

var (
	SelectedStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Primary).
			Bold(true)

	HoverStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(SurfaceAlt)

	CursorStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	LinkStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Underline(true)
)

var (
	StatusStyle = lipgloss.NewStyle().
			Foreground(MutedText).
			Background(Surface)

	SuccessStatus = StatusStyle.Copy().Foreground(Success)
	ErrorStatus   = StatusStyle.Copy().Foreground(Error)
	WarningStatus = StatusStyle.Copy().Foreground(Warning)
)

var (
	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedItemStyle = ListItemStyle.Copy().
				Foreground(White).
				Background(Primary)

	HoverItemStyle = ListItemStyle.Copy().
			Foreground(White).
			Background(SurfaceAlt)

	ListItemTitleStyle = lipgloss.NewStyle().
				Bold(true)

	ListItemDescStyle = lipgloss.NewStyle().
				Foreground(Secondary)

	ListItemMetaStyle = lipgloss.NewStyle().
				Foreground(MutedText)
)

var (
	TagStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Background(Surface).
			Padding(0, 1).
			MarginRight(1)

	TagSelectedStyle = TagStyle.Copy().
				Foreground(White).
				Background(Accent)

	TagHoverStyle = TagStyle.Copy().
			Foreground(White).
			Background(SurfaceAlt)
)

var (
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(Border).
			Padding(1, 2).
			Margin(1)

	PanelStyle = lipgloss.NewStyle().
			Background(Surface).
			Padding(1)

	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(2).
			Background(Background)
)

var (
	InputStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Surface).
			Border(lipgloss.NormalBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	InputFocusedStyle = InputStyle.Copy().
				BorderForeground(Primary)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Padding(0, 1).
			MarginRight(1)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(MutedText)
)

var (
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Secondary).
				Background(Surface)

	TableRowStyle = lipgloss.NewStyle().
			Foreground(Text)

	TableRowAltStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(SurfaceAlt)

	TableSelectedStyle = lipgloss.NewStyle().
				Foreground(White).
				Background(Primary)
)
