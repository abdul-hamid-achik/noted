// Package layout computes responsive screen regions (in terminal cell coordinates) for the
// noted TUI. It is the single source of truth for where panes live, so both rendering and mouse
// hit-testing read from the same rectangles. See docs/dev/charm-v2-reference.md.
package layout

// Rect is a region in 0-based terminal cell coordinates (top-left origin).
type Rect struct {
	X, Y, W, H int
}

// Contains reports whether (x,y) falls inside the rect.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Empty reports whether the rect has no drawable area.
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Mode signals whether the terminal is usable or too small to render the UI.
type Mode int

const (
	ModeNormal Mode = iota
	ModeTooSmall
)

// Thresholds (cells).
const (
	// MinWidth/MinHeight: below either, we show a "terminal too small" notice instead of the UI.
	MinWidth  = 40
	MinHeight = 12

	// Below SidebarCollapseBelow the sidebar is hidden (toggle-on-demand) to give content room.
	SidebarCollapseBelow = 70

	sidebarMin = 22
	sidebarMax = 34

	footerHeight = 1
)

// Options describes what the active view wants from the layout.
type Options struct {
	// ShowSidebar: the view has a left sidebar/nav. Ignored in ModeTooSmall.
	ShowSidebar bool
	// SidebarForced keeps the sidebar visible even on narrow terminals (e.g. user toggled it on).
	SidebarForced bool
	// HeaderHeight reserves N rows at the top for a header/tab bar (0 = none).
	HeaderHeight int
}

// Regions is the computed set of panes for one frame.
type Regions struct {
	Mode           Mode
	Full           Rect // entire screen
	Header         Rect // top bar (Empty if HeaderHeight == 0)
	Sidebar        Rect // left nav (Empty if collapsed/disabled)
	Main           Rect // primary content
	Footer         Rect // bottom status/help line
	SidebarVisible bool
}

// sidebarWidth returns the responsive sidebar width for a given total width.
func sidebarWidth(w int) int {
	sw := w / 4
	if sw < sidebarMin {
		sw = sidebarMin
	}
	if sw > sidebarMax {
		sw = sidebarMax
	}
	return sw
}

// Compute returns the regions for a (width,height) terminal and the view's options.
func Compute(w, h int, opt Options) Regions {
	full := Rect{X: 0, Y: 0, W: w, H: h}
	if w < MinWidth || h < MinHeight {
		return Regions{Mode: ModeTooSmall, Full: full}
	}

	r := Regions{Mode: ModeNormal, Full: full}

	// Footer pinned to the bottom row(s).
	r.Footer = Rect{X: 0, Y: h - footerHeight, W: w, H: footerHeight}

	// Body is everything above the footer.
	bodyY := 0
	bodyH := h - footerHeight

	// Header at the top of the body.
	if opt.HeaderHeight > 0 {
		hh := opt.HeaderHeight
		if hh > bodyH-1 {
			hh = bodyH - 1
		}
		r.Header = Rect{X: 0, Y: bodyY, W: w, H: hh}
		bodyY += hh
		bodyH -= hh
	}

	// Decide sidebar visibility.
	showSidebar := opt.ShowSidebar && (opt.SidebarForced || w >= SidebarCollapseBelow)
	r.SidebarVisible = showSidebar

	if showSidebar {
		sw := sidebarWidth(w)
		if sw > w-sidebarMin { // guarantee main keeps a minimum width
			sw = w - sidebarMin
		}
		r.Sidebar = Rect{X: 0, Y: bodyY, W: sw, H: bodyH}
		r.Main = Rect{X: sw, Y: bodyY, W: w - sw, H: bodyH}
	} else {
		r.Main = Rect{X: 0, Y: bodyY, W: w, H: bodyH}
	}

	return r
}
