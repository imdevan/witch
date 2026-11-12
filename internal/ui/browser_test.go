package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/imdevan/witch/internal/domain"
)

func testTheme() Theme { return ThemeFromConfig(domain.DefaultConfig()) }

// fixtureEntries writes a small alias file and returns entries pointing at it.
func fixtureEntries(t *testing.T) []domain.Entry {
	t.Helper()
	dir := t.TempDir()
	aliases := filepath.Join(dir, "aliases.zsh")
	var lines []string
	for i := 1; i <= 60; i++ {
		lines = append(lines, "# filler line")
	}
	lines[39] = `alias asdf="cowsay 'hi'"`
	lines[49] = "greet() {"
	lines[50] = "\techo hi"
	lines[51] = "}"
	if err := os.WriteFile(aliases, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F', 0, 0, 0}, 0o755); err != nil {
		t.Fatal(err)
	}
	return []domain.Entry{
		{Name: "asdf", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: "cowsay 'hi'", Definition: `alias asdf="cowsay 'hi'"`, File: aliases, Line: 40},
		{Name: "greet", Type: domain.TypeFunction, Value: "greet() {\n\techo hi\n}", Definition: "greet() {\n\techo hi\n}", File: aliases, Line: 50},
		{Name: "tool", Type: domain.TypeExecutable, File: bin},
	}
}

func press(m tea.Model, keys ...string) tea.Model {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "shift+tab":
			msg = tea.KeyMsg{Type: tea.KeyShiftTab}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "ctrl+c":
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		m, _ = m.Update(msg)
	}
	return m
}

func sized(m browserModel, w, h int) browserModel {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(browserModel)
}

func TestBrowserSplitsAtLargeBreakpoint(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	if !m.split() {
		t.Fatal("expected split layout at 120 columns")
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"asdf", "aliases.zsh:40", `40 ▌alias asdf="cowsay 'hi'"`} {
		if !strings.Contains(view, want) {
			t.Errorf("split view missing %q:\n%s", want, view)
		}
	}
	// Both panes sit side by side on the same rows.
	if !strings.Contains(view, "╮╭") {
		t.Errorf("expected two panes joined horizontally:\n%s", view)
	}
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > 120 {
			t.Errorf("line %d is %d wide, exceeds terminal width", i, w)
		}
	}
	if h := lipgloss.Height(m.View()); h > 29 {
		t.Errorf("view is %d lines, should fit inline above the prompt", h)
	}
}

func TestBrowserNarrowListThenPreview(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 70, 30)
	if m.split() {
		t.Fatal("expected list-only layout below BreakpointLG")
	}
	if view := ansi.Strip(m.View()); strings.Contains(view, "▌alias asdf") {
		t.Fatalf("narrow list should not show the preview:\n%s", view)
	}

	next := press(m, "enter").(browserModel)
	if !next.previewMode || next.quitting {
		t.Fatal("enter should open the preview on narrow screens")
	}
	if view := ansi.Strip(next.View()); !strings.Contains(view, `40 ▌alias asdf="cowsay 'hi'"`) {
		t.Fatalf("expected preview view:\n%s", view)
	}

	back := press(next, "esc").(browserModel)
	if back.previewMode || back.quitting {
		t.Fatal("esc should return to the list")
	}

	done := press(next, "enter").(browserModel)
	if !done.quitting || done.result.Action != ActionPrint || done.result.Entry.Name != "asdf" {
		t.Fatalf("enter in preview should select, got %+v", done.result)
	}
}

func TestBrowserEnterPrintsSelection(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	done := press(m, "down", "enter").(browserModel)
	if !done.quitting || done.result.Action != ActionPrint || done.result.Entry == nil || done.result.Entry.Name != "greet" {
		t.Fatalf("got %+v", done.result)
	}
	if done.View() != "" {
		t.Error("view should clear on exit")
	}
}

func TestBrowserQuit(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
		done := press(m, k).(browserModel)
		if !done.quitting || done.result.Action != ActionNone {
			t.Errorf("%s: expected quit without selection, got %+v", k, done.result)
		}
	}
}

func TestBrowserPrefilledFilter(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{Filter: "gre"}), 120, 30)
	if got := len(m.list.VisibleItems()); got != 1 {
		t.Fatalf("expected 1 match for 'gre', got %d", got)
	}
	if e := m.preview.Entry(); e == nil || e.Name != "greet" {
		t.Fatalf("preview should follow the filtered selection, got %+v", e)
	}
	// esc clears the applied filter instead of quitting.
	cleared := press(m, "esc").(browserModel)
	if cleared.quitting || len(cleared.list.VisibleItems()) != 3 {
		t.Fatalf("esc should clear the filter, quitting=%v items=%d", cleared.quitting, len(cleared.list.VisibleItems()))
	}
}

func TestBrowserPassesFilterToPreview(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{Filter: "gre"}), 120, 30)
	if m.preview.query != "gre" {
		t.Fatalf("preview query = %q, want the CLI filter", m.preview.query)
	}
	cleared := press(m, "esc").(browserModel)
	if cleared.preview.query != "" {
		t.Fatalf("clearing the filter should clear the preview query, got %q", cleared.preview.query)
	}
	typing := press(cleared, "/", "a", "s").(browserModel)
	if typing.preview.query != "as" {
		t.Fatalf("typing a filter should update the preview query, got %q", typing.preview.query)
	}
}

func TestBrowserTypeCycle(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	want := []struct {
		filter TypeFilter
		names  []string
	}{
		{FilterAlias, []string{"asdf"}},
		{FilterFunction, []string{"greet"}},
		{FilterExecutable, []string{"tool"}},
		{FilterAll, []string{"asdf", "greet", "tool"}},
	}
	var model tea.Model = m
	for _, w := range want {
		model = press(model, "tab")
		bm := model.(browserModel)
		if bm.typeFilter != w.filter {
			t.Fatalf("filter = %v, want %v", bm.typeFilter, w.filter)
		}
		var names []string
		for _, item := range bm.list.Items() {
			names = append(names, item.(EntryItem).Entry.Name)
		}
		if strings.Join(names, ",") != strings.Join(w.names, ",") {
			t.Errorf("%s: items = %v, want %v", w.filter.Label(), names, w.names)
		}
		if w.filter != FilterAll && !strings.Contains(bm.list.Title, w.filter.Label()) {
			t.Errorf("title %q should name the filter", bm.list.Title)
		}
	}
	back := press(model, "shift+tab").(browserModel)
	if back.typeFilter != FilterExecutable {
		t.Errorf("shift+tab should cycle backwards, got %v", back.typeFilter)
	}
}

func TestBrowserCopy(t *testing.T) {
	var copied []string
	copyFn := func(s string) error { copied = append(copied, s); return nil }
	entries := fixtureEntries(t)
	m := sized(newBrowserModel(entries, testTheme(), BrowserOptions{Copy: copyFn}), 120, 30)

	m = press(m, "y").(browserModel)
	m = press(m, "Y").(browserModel)
	m = press(m, "c").(browserModel)
	want := []string{`alias asdf="cowsay 'hi'"`, entries[0].File + ":40", "asdf"}
	if strings.Join(copied, "|") != strings.Join(want, "|") {
		t.Fatalf("copied %q, want %q", copied, want)
	}
	if m.quitting || !strings.Contains(m.status, "copied name") {
		t.Errorf("copy should stay open with a status, got quitting=%v status=%q", m.quitting, m.status)
	}

	// Executables copy their path.
	copied = nil
	m = press(m, "down", "down", "y").(browserModel)
	if len(copied) != 1 || copied[0] != entries[2].File {
		t.Errorf("executable copy = %q", copied)
	}

	failing := sized(newBrowserModel(entries, testTheme(), BrowserOptions{Copy: func(string) error { return errors.New("no clipboard") }}), 120, 30)
	if got := press(failing, "y").(browserModel).status; !strings.Contains(got, "no clipboard") {
		t.Errorf("status = %q", got)
	}
}

func TestBrowserOpen(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	done := press(m, "o").(browserModel)
	if !done.quitting || done.result.Action != ActionOpen || done.result.Entry.Name != "asdf" {
		t.Fatalf("got %+v", done.result)
	}

	exe := press(m, "down", "down", "o").(browserModel)
	if exe.quitting || !strings.Contains(exe.status, "executable") {
		t.Fatalf("o on an executable should stay open with a status, got quitting=%v %q", exe.quitting, exe.status)
	}
}

func TestBrowserResponsiveHelp(t *testing.T) {
	entries := fixtureEntries(t)
	widths := map[int]int{30: 2, 50: 3, 70: 4}
	for width, want := range widths {
		m := sized(newBrowserModel(entries, testTheme(), BrowserOptions{}), width, 30)
		keys := m.shortHelpKeys()
		if len(keys) != want {
			t.Errorf("width %d: %d short help keys, want %d", width, len(keys), want)
		}
		if keys[len(keys)-1].Help().Key != "?" {
			t.Errorf("width %d: help toggle should always be last", width)
		}
	}
	wide := sized(newBrowserModel(entries, testTheme(), BrowserOptions{}), 120, 30)
	if len(wide.shortHelpKeys()) != 7 {
		t.Errorf("wide screens should show all short keys, got %d", len(wide.shortHelpKeys()))
	}

	full := press(wide, "?").(browserModel)
	if !full.help.ShowAll || !strings.Contains(ansi.Strip(full.View()), "copy path:line") {
		t.Error("? should show full help")
	}
	if lipgloss.Height(full.View()) > 29 {
		t.Error("full help should shrink panes to keep the view inline-sized")
	}
}

func TestBrowserPreviewScroll(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	start := m.preview.YOffset()
	m = press(m, "J", "J").(browserModel)
	if m.preview.YOffset() != start+2 {
		t.Errorf("J should scroll the preview: %d -> %d", start, m.preview.YOffset())
	}
	m = press(m, "K").(browserModel)
	if m.preview.YOffset() != start+1 {
		t.Errorf("K should scroll up: got %d", m.preview.YOffset())
	}
	if m.list.Index() != 0 {
		t.Error("preview scrolling should not move the list selection")
	}
}
