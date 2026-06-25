// Package theme defines the Nord color palette and component theming helpers for the
// noted TUI. Nord (dark) is the default and only theme today. See docs/dev/charm-v2-reference.md.
package theme

import "charm.land/lipgloss/v2"

// Nord palette — https://www.nordtheme.com/
var (
	// Polar Night — dark backgrounds.
	Nord0 = lipgloss.Color("#2E3440") // base background
	Nord1 = lipgloss.Color("#3B4252") // elevated surface
	Nord2 = lipgloss.Color("#434C5E") // higher surface / selection bg
	Nord3 = lipgloss.Color("#4C566A") // borders / muted / disabled

	// Snow Storm — light foregrounds.
	Nord4 = lipgloss.Color("#D8DEE9") // muted text
	Nord5 = lipgloss.Color("#E5E9F0") // subtle text
	Nord6 = lipgloss.Color("#ECEFF4") // primary text

	// Frost — primary accents (cool blues/teal).
	Nord7  = lipgloss.Color("#8FBCBB") // teal
	Nord8  = lipgloss.Color("#88C0D0") // primary accent (cyan)
	Nord9  = lipgloss.Color("#81A1C1") // secondary accent (blue)
	Nord10 = lipgloss.Color("#5E81AC") // deep blue

	// Aurora — semantic / status colors.
	Nord11 = lipgloss.Color("#BF616A") // red    — error
	Nord12 = lipgloss.Color("#D08770") // orange — warning-alt
	Nord13 = lipgloss.Color("#EBCB8B") // yellow — warning / highlight
	Nord14 = lipgloss.Color("#A3BE8C") // green  — success
	Nord15 = lipgloss.Color("#B48EAD") // purple — special
)

// Semantic tokens — code should prefer these over raw Nord* so a future light theme is a
// single-file change.
var (
	Bg        = Nord0 // app background
	Surface   = Nord1 // panels / cards
	SurfaceHi = Nord2 // selection / hover background
	Border    = Nord3 // borders, separators
	Muted     = Nord4 // secondary text
	Subtle    = Nord5 // tertiary text
	Text      = Nord6 // primary text

	Primary   = Nord8  // primary accent (titles, active)
	Secondary = Nord9  // secondary accent
	Accent    = Nord10 // deep accent
	Link      = Nord8  // wikilinks / hyperlinks

	Success = Nord14
	Warning = Nord13
	ErrorC  = Nord11
	Info    = Nord8

	Selection = Nord2 // selected-row background
)
