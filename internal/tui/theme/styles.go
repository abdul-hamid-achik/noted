package theme

import "charm.land/lipgloss/v2"

// Reusable lipgloss styles for the rewrite. All derive from the Nord semantic tokens.
var (
	Base = lipgloss.NewStyle().Foreground(Text).Background(Bg)

	Title = lipgloss.NewStyle().Foreground(Primary).Bold(true)

	Heading = lipgloss.NewStyle().Foreground(Text).Bold(true)

	Subheading = lipgloss.NewStyle().Foreground(Secondary)

	MutedStyle = lipgloss.NewStyle().Foreground(Muted)

	// Panel is an elevated surface (sidebar, cards).
	Panel = lipgloss.NewStyle().Background(Surface).Foreground(Text)

	// Card is a bordered box.
	Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	// Modal is a centered, accented overlay (palette, switcher, help).
	Modal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Background(Surface).
		Padding(1, 2)

	// Footer / status bar.
	Footer = lipgloss.NewStyle().Foreground(Muted).Background(Surface)

	// Active vs inactive pane borders.
	PaneActive   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(Primary)
	PaneInactive = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(Border)

	// Key hints in footers/help.
	HelpKey  = lipgloss.NewStyle().Foreground(Primary)
	HelpDesc = lipgloss.NewStyle().Foreground(Muted)

	// Selected row (when not using a component's own selection style).
	Selected = lipgloss.NewStyle().Foreground(Text).Background(Selection).Bold(true)

	// Wikilink rendering.
	LinkStyle = lipgloss.NewStyle().Foreground(Link).Underline(true)
)
