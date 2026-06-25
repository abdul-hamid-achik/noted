package tui

import "testing"

func TestToggleTaskLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		completed bool
		want      string
	}{
		{"mark dash done", "- [ ] buy milk", false, "- [x] buy milk"},
		{"mark star done", "* [ ] buy milk", false, "* [x] buy milk"},
		{"unmark lowercase", "- [x] buy milk", true, "- [ ] buy milk"},
		{"unmark uppercase", "- [X] buy milk", true, "- [ ] buy milk"},
		{"only first checkbox marked", "- [ ] a then [ ] b", false, "- [x] a then [ ] b"},
		{"only first checkbox unmarked", "- [x] a then [x] b", true, "- [ ] a then [x] b"},
		{"no checkbox unchanged when marking", "just a line", false, "just a line"},
		{"no checkbox unchanged when unmarking", "just a line", true, "just a line"},
		{"indented checkbox", "  - [ ] nested", false, "  - [x] nested"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toggleTaskLine(tt.line, tt.completed); got != tt.want {
				t.Errorf("toggleTaskLine(%q, %v) = %q, want %q", tt.line, tt.completed, got, tt.want)
			}
		})
	}
}
