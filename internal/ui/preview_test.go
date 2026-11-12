package ui

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

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

func TestMatchIndices(t *testing.T) {
	tests := []struct {
		line, name, query string
		want              []int
	}{
		{`alias asdf="cowsay 'hi'"`, "asdf", "", []int{6, 7, 8, 9}},
		// whole-word only for the name
		{"asdfx asdf x_asdf asdf-y", "asdf", "", []int{6, 7, 8, 9}},
		{"cd ../..", "..", "", []int{3, 4, 6, 7}},
		// query is a case-insensitive substring, in addition to the name
		{"greet() { Greeting }", "greet", "GRE", []int{0, 1, 2, 3, 4, 10, 11, 12}},
		// query equal to the name adds nothing
		{"greetings greet", "greet", "Greet", []int{10, 11, 12, 13, 14}},
		{"nothing here", "asdf", "zz", nil},
		// rune indices, not byte indices
		{"é asdf", "asdf", "", []int{2, 3, 4, 5}},
	}
	for _, tt := range tests {
		got := MatchIndices(tt.line, tt.name, tt.query)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("MatchIndices(%q, %q, %q) = %v, want %v", tt.line, tt.name, tt.query, got, tt.want)
		}
	}
}

func TestPreviewHighlightsNameAndQuery(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	theme := testTheme()
	theme.Highlight = lipgloss.Color("3")
	entries := fixtureEntries(t)
	p := NewPreview(theme)
	p.SetSize(80, 20)
	p.SetEntry(&entries[0])

	// Background color 3 (from the theme) with a dark foreground for contrast.
	bg := regexp.MustCompile(`\x1b\[[0-9;]*\b30;43m`)
	line := previewLine(t, p, 40)
	if !bg.MatchString(line) {
		t.Fatalf("expected the name to use the highlight background, got %q", line)
	}
	if got := ansi.Strip(line); !strings.Contains(got, `alias asdf="cowsay 'hi'"`) {
		t.Fatalf("highlighting should not change the text, got %q", got)
	}
	if bg.MatchString(previewLine(t, p, 39)) {
		t.Error("lines without matches should not be highlighted")
	}

	offset := p.YOffset()
	p.SetQuery("filler")
	if !bg.MatchString(previewLine(t, p, 39)) {
		t.Error("search term should be highlighted on other lines")
	}
	if p.YOffset() != offset {
		t.Error("changing the query should keep the scroll position")
	}
}

// previewLine returns the rendered content row for a 1-indexed file line.
func previewLine(t *testing.T, p Preview, n int) string {
	t.Helper()
	for _, row := range strings.Split(p.viewport.View(), "\n") {
		if strings.HasPrefix(strings.TrimSpace(ansi.Strip(row)), fmt.Sprint(n)+" ") {
			return row
		}
	}
	t.Fatalf("line %d not visible", n)
	return ""
}

func TestPreviewShowsExecutableHelp(t *testing.T) {
	entries := fixtureEntries(t)
	p := NewPreview(testTheme())
	p.SetSize(60, 15)
	p.EnableHelp()
	p.SetEntry(&entries[2])

	if path, needed := p.NeedsHelp(); !needed || path != entries[2].File {
		t.Fatalf("NeedsHelp() = %q, %v", path, needed)
	}
	if view := ansi.Strip(p.View()); !strings.Contains(view, "loading help") {
		t.Errorf("expected a loading state:\n%s", view)
	}

	p.SetHelp(entries[2].File, HelpPage{Source: "man tool", Text: "NAME\n\ttool - does things"}, nil)
	view := ansi.Strip(p.View())
	for _, want := range []string{"man tool", "NAME", "    tool - does things"} {
		if !strings.Contains(view, want) {
			t.Errorf("help preview missing %q:\n%s", want, view)
		}
	}
	if _, needed := p.NeedsHelp(); needed {
		t.Error("help should be cached")
	}

	// Aliases and functions keep the file preview.
	p.SetEntry(&entries[0])
	if _, needed := p.NeedsHelp(); needed {
		t.Error("aliases should not need help")
	}
	if view := ansi.Strip(p.View()); !strings.Contains(view, `40 ▌alias asdf="cowsay 'hi'"`) {
		t.Errorf("alias preview changed:\n%s", view)
	}
}

func TestPreviewHelpErrorFallsBack(t *testing.T) {
	entries := fixtureEntries(t)
	p := NewPreview(testTheme())
	p.SetSize(60, 15)
	p.EnableHelp()
	p.SetEntry(&entries[2])
	p.SetHelp(entries[2].File, HelpPage{}, fmt.Errorf("no man page or --help output"))
	if view := ansi.Strip(p.View()); !strings.Contains(view, "binary file, not shown") {
		t.Errorf("expected the file details fallback:\n%s", view)
	}
}
