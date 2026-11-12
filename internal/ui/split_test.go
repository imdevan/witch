package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestShouldSplit(t *testing.T) {
	for bp, want := range map[Breakpoint]bool{
		BreakpointXS: false, BreakpointSM: false, BreakpointMD: false,
		BreakpointLG: true, BreakpointXL: true,
	} {
		if got := ShouldSplit(bp); got != want {
			t.Errorf("ShouldSplit(%v) = %v, want %v", bp, got, want)
		}
	}
}

func TestSplitLayoutRender(t *testing.T) {
	for _, width := range []int{80, 100, 160, 240} {
		s := NewSplitLayout(testTheme())
		s.Width, s.Height = width, 12
		lw, rw, h := s.ContentSizes()
		left := strings.Repeat(strings.Repeat("L", lw)+"\n", h-1) + strings.Repeat("L", lw)
		right := strings.Repeat(strings.Repeat("R", rw)+"\n", h+5) // taller than the pane
		out := s.Render(left, right)
		if got := lipgloss.Width(out); got != width {
			t.Errorf("width %d: rendered width %d", width, got)
		}
		if got := lipgloss.Height(out); got != 12 {
			t.Errorf("width %d: rendered height %d, want 12", width, got)
		}
		if lw < 30-4 || lw > 60 {
			t.Errorf("width %d: left content width %d outside clamps", width, lw)
		}
	}

	s := NewSplitLayout(testTheme())
	s.Width, s.Height = 50, 8
	out := s.RenderSingle("hello")
	if lipgloss.Width(out) != 50 || lipgloss.Height(out) != 8 {
		t.Errorf("single pane is %dx%d, want 50x8", lipgloss.Width(out), lipgloss.Height(out))
	}
}
