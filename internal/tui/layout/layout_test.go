package layout

import "testing"

func TestTooSmall(t *testing.T) {
	for _, tc := range []struct{ w, h int }{{10, 10}, {39, 20}, {120, 11}} {
		if got := Compute(tc.w, tc.h, Options{ShowSidebar: true}); got.Mode != ModeTooSmall {
			t.Errorf("Compute(%d,%d) mode = %v, want ModeTooSmall", tc.w, tc.h, got.Mode)
		}
	}
}

func TestRegionsCoverScreenNoOverlap(t *testing.T) {
	r := Compute(120, 40, Options{ShowSidebar: true, HeaderHeight: 2})
	if r.Mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", r.Mode)
	}
	// Footer pinned to bottom.
	if r.Footer.Y != 39 || r.Footer.H != 1 {
		t.Errorf("footer = %+v, want bottom row", r.Footer)
	}
	// Header spans the top.
	if r.Header.Y != 0 || r.Header.H != 2 || r.Header.W != 120 {
		t.Errorf("header = %+v", r.Header)
	}
	// Sidebar + main are side by side, below the header, above the footer.
	if !r.SidebarVisible || r.Sidebar.Empty() {
		t.Fatalf("sidebar should be visible at width 120")
	}
	if r.Sidebar.X != 0 || r.Sidebar.Y != 2 {
		t.Errorf("sidebar origin = %+v, want (0,2)", r.Sidebar)
	}
	if r.Main.X != r.Sidebar.W {
		t.Errorf("main.X = %d, want sidebar.W = %d (no gap/overlap)", r.Main.X, r.Sidebar.W)
	}
	if r.Sidebar.W+r.Main.W != 120 {
		t.Errorf("sidebar.W+main.W = %d, want 120 (cover width)", r.Sidebar.W+r.Main.W)
	}
	if r.Main.Y+r.Main.H != 39 {
		t.Errorf("main bottom = %d, want 39 (meets footer)", r.Main.Y+r.Main.H)
	}
}

func TestSidebarCollapsesWhenNarrow(t *testing.T) {
	r := Compute(64, 24, Options{ShowSidebar: true})
	if r.SidebarVisible {
		t.Errorf("sidebar should collapse below width %d", SidebarCollapseBelow)
	}
	if r.Main.W != 64 {
		t.Errorf("main should span full width when sidebar collapsed, got W=%d", r.Main.W)
	}
}

func TestSidebarForcedStaysVisible(t *testing.T) {
	r := Compute(64, 24, Options{ShowSidebar: true, SidebarForced: true})
	if !r.SidebarVisible {
		t.Errorf("forced sidebar should stay visible")
	}
	if r.Main.W < sidebarMin {
		t.Errorf("main width %d should remain >= %d", r.Main.W, sidebarMin)
	}
}

// TestThresholdBoundaries pins the exact MinWidth/MinHeight edges: at the threshold the UI renders,
// one cell under either dimension flips to ModeTooSmall.
func TestThresholdBoundaries(t *testing.T) {
	if got := Compute(MinWidth, MinHeight, Options{ShowSidebar: true}); got.Mode != ModeNormal {
		t.Errorf("Compute(%d,%d) at threshold = %v, want ModeNormal", MinWidth, MinHeight, got.Mode)
	}
	if got := Compute(MinWidth-1, MinHeight, Options{}); got.Mode != ModeTooSmall {
		t.Errorf("one cell under MinWidth = %v, want ModeTooSmall", got.Mode)
	}
	if got := Compute(MinWidth, MinHeight-1, Options{}); got.Mode != ModeTooSmall {
		t.Errorf("one cell under MinHeight = %v, want ModeTooSmall", got.Mode)
	}
}

// TestSidebarCollapseBoundary pins the exact SidebarCollapseBelow edge.
func TestSidebarCollapseBoundary(t *testing.T) {
	if r := Compute(SidebarCollapseBelow, 24, Options{ShowSidebar: true}); !r.SidebarVisible {
		t.Errorf("at width %d the sidebar should be visible", SidebarCollapseBelow)
	}
	if r := Compute(SidebarCollapseBelow-1, 24, Options{ShowSidebar: true}); r.SidebarVisible {
		t.Errorf("at width %d the sidebar should collapse", SidebarCollapseBelow-1)
	}
}

// TestForcedSidebarNarrowKeepsMain checks the main-width guarantee at the smallest renderable width:
// a forced sidebar must never shrink main below sidebarMin, and the two must still tile the width.
func TestForcedSidebarNarrowKeepsMain(t *testing.T) {
	r := Compute(MinWidth, MinHeight, Options{ShowSidebar: true, SidebarForced: true})
	if !r.SidebarVisible {
		t.Fatal("forced sidebar should be visible even at MinWidth")
	}
	if r.Main.W < sidebarMin {
		t.Errorf("main width %d < sidebarMin %d", r.Main.W, sidebarMin)
	}
	if r.Sidebar.X != 0 || r.Main.X != r.Sidebar.W {
		t.Errorf("panes overlap/gap: sidebar=%+v main=%+v", r.Sidebar, r.Main)
	}
	if r.Sidebar.W+r.Main.W != MinWidth {
		t.Errorf("sidebar.W+main.W = %d, want %d (cover width)", r.Sidebar.W+r.Main.W, MinWidth)
	}
}

// TestNoHeaderLeavesBodyAtTop: with HeaderHeight 0 the header is empty and the body starts at row 0.
func TestNoHeaderLeavesBodyAtTop(t *testing.T) {
	r := Compute(120, 40, Options{ShowSidebar: true})
	if !r.Header.Empty() {
		t.Errorf("header should be empty when HeaderHeight=0, got %+v", r.Header)
	}
	if r.Sidebar.Y != 0 || r.Main.Y != 0 {
		t.Errorf("body should start at row 0, sidebar.Y=%d main.Y=%d", r.Sidebar.Y, r.Main.Y)
	}
}

// TestCoverInvariantAcrossSizes is a property check: across many normal-mode sizes the footer is the
// bottom row, sidebar+main tile the full width with no overlap, and the body meets the footer.
func TestCoverInvariantAcrossSizes(t *testing.T) {
	for w := MinWidth; w <= 200; w += 7 {
		for h := MinHeight; h <= 60; h += 5 {
			r := Compute(w, h, Options{ShowSidebar: true, HeaderHeight: 1})
			if r.Mode != ModeNormal {
				t.Fatalf("Compute(%d,%d) = %v, want ModeNormal", w, h, r.Mode)
			}
			if r.Footer.Y != h-1 || r.Footer.W != w {
				t.Errorf("(%d,%d) footer = %+v, want bottom row spanning width", w, h, r.Footer)
			}
			if r.Main.X != r.Sidebar.W {
				t.Errorf("(%d,%d) gap/overlap: main.X=%d sidebar.W=%d", w, h, r.Main.X, r.Sidebar.W)
			}
			if r.Sidebar.W+r.Main.W != w {
				t.Errorf("(%d,%d) panes width sum = %d, want %d", w, h, r.Sidebar.W+r.Main.W, w)
			}
			if r.Main.Y+r.Main.H != h-1 {
				t.Errorf("(%d,%d) main bottom = %d, want %d (meets footer)", w, h, r.Main.Y+r.Main.H, h-1)
			}
		}
	}
}
