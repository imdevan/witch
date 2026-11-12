package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/imdevan/witch/internal/domain"
)

func TestPreviewHighlightsDefinition(t *testing.T) {
	entries := fixtureEntries(t)
	p := NewPreview(testTheme())
	p.SetSize(60, 15)

	p.SetEntry(&entries[1]) // greet(), lines 50-52
	if p.HighlightStart != 50 || p.HighlightEnd != 52 {
		t.Fatalf("highlight = %d-%d, want 50-52", p.HighlightStart, p.HighlightEnd)
	}
	view := ansi.Strip(p.View())
	for _, want := range []string{"greet  function", "50 ▌greet() {", "51 ▌    echo hi", "52 ▌}"} {
		if !strings.Contains(view, want) {
			t.Errorf("preview missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "49 ▌") || strings.Contains(view, "53 ▌") {
		t.Errorf("only the definition should be highlighted:\n%s", view)
	}
	// The definition is scrolled into view with some context above it.
	if off := p.YOffset(); off <= 0 || off >= 49 {
		t.Errorf("yoffset = %d, expected the definition near the top", off)
	}
	if h := lipgloss.Height(p.View()); h != 15 {
		t.Errorf("preview height = %d, want 15", h)
	}

	p.SetEntry(&entries[0]) // single-line alias
	if p.HighlightStart != 40 || p.HighlightEnd != 40 {
		t.Fatalf("alias highlight = %d-%d", p.HighlightStart, p.HighlightEnd)
	}
}

func TestPreviewCachesFiles(t *testing.T) {
	entries := fixtureEntries(t)
	p := NewPreview(testTheme())
	reads := 0
	p.readFile = func(path string) ([]string, error) {
		reads++
		return readTextLines(path)
	}
	p.SetSize(60, 15)
	p.SetEntry(&entries[0])
	p.SetEntry(&entries[1])
	p.SetEntry(&entries[0])
	if reads != 1 {
		t.Errorf("file read %d times, want 1", reads)
	}
}

func TestPreviewBinaryShowsDetails(t *testing.T) {
	entries := fixtureEntries(t)
	p := NewPreview(testTheme())
	p.SetSize(60, 15)
	p.SetEntry(&entries[2])
	view := ansi.Strip(p.View())
	for _, want := range []string{"tool  executable", "path", "binary file, not shown", "-rwxr-xr-x"} {
		if !strings.Contains(view, want) {
			t.Errorf("binary preview missing %q:\n%s", want, view)
		}
	}
}

func TestPreviewEmpty(t *testing.T) {
	p := NewPreview(testTheme())
	p.SetSize(40, 10)
	p.SetEntry(nil)
	if !strings.Contains(ansi.Strip(p.View()), "Nothing selected") {
		t.Error("expected empty state")
	}
	p.SetEntry(&domain.Entry{Name: "gone", Type: domain.TypeAlias, File: "/does/not/exist", Line: 3})
	if !strings.Contains(ansi.Strip(p.View()), "file not found") {
		t.Errorf("expected a read error in the preview:\n%s", ansi.Strip(p.View()))
	}
}
