package tui

import (
	"strings"
	"testing"
)

// TestRenderMarkdownNordTheme proves the markdown preview uses the Nord Glamour style: an H1
// renders with the Nord8 (#88c0d0 = rgb 136,192,208) accent background as a truecolor SGR.
func TestRenderMarkdownNordTheme(t *testing.T) {
	out := renderMarkdown("# Hello", 60)
	if !strings.Contains(out, "136;192;208") {
		t.Errorf("expected Nord8 (136;192;208) in rendered H1, got:\n%q", out)
	}
}

func TestRenderMarkdownEmptyIsEmpty(t *testing.T) {
	if got := renderMarkdown("   \n  ", 60); got != "" {
		t.Errorf("blank content should render empty, got %q", got)
	}
}
