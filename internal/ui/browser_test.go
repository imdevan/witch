package ui

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

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
	special := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "shift+tab": tea.KeyShiftTab,
		"down": tea.KeyDown, "up": tea.KeyUp, "ctrl+c": tea.KeyCtrlC, "backspace": tea.KeyBackspace,
		"shift+down": tea.KeyShiftDown, "shift+up": tea.KeyShiftUp, "ctrl+o": tea.KeyCtrlO,
		"ctrl+y": tea.KeyCtrlY, "ctrl+n": tea.KeyCtrlN, "ctrl+p": tea.KeyCtrlP,
		"ctrl+d": tea.KeyCtrlD, "ctrl+u": tea.KeyCtrlU,
	}
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if t, ok := special[k]; ok {
			msg = tea.KeyMsg{Type: t}
		}
		m, _ = m.Update(msg)
	}
	return m
}

// typeText sends each rune as its own keystroke, like a user typing.
func typeText(m tea.Model, text string) tea.Model {
	for _, r := range text {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
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
	if !done.quitting || done.result.Action != ActionSelect || done.result.Entry.Name != "asdf" {
		t.Fatalf("enter in preview should select, got %+v", done.result)
	}
}

func TestBrowserStartsInSearch(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	if m.focus != focusSearch || !m.search.Focused() {
		t.Fatal("browser should start with the search focused")
	}
	view := ansi.Strip(m.View())
	if strings.Contains(view, "Filter:") || strings.Contains(view, "filtered") {
		t.Errorf("list filter label/status should be hidden:\n%s", view)
	}

	// Letters are typed into the search, not treated as shortcuts.
	typed := typeText(m, "qyo").(browserModel)
	if typed.quitting || typed.search.Value() != "qyo" {
		t.Fatalf("typing should search, got quitting=%v value=%q", typed.quitting, typed.search.Value())
	}

	// Each keystroke filters the list and moves the preview.
	gre := typeText(m, "gre").(browserModel)
	if len(gre.list.VisibleItems()) != 1 || gre.preview.Entry().Name != "greet" {
		t.Fatalf("expected greet to be the only match, got %d items, preview %v", len(gre.list.VisibleItems()), gre.preview.Entry())
	}
	cleared := press(gre, "backspace", "backspace", "backspace").(browserModel)
	if len(cleared.list.VisibleItems()) != 3 || cleared.list.FilterState() != list.Unfiltered {
		t.Fatalf("clearing the search should show everything, got %d", len(cleared.list.VisibleItems()))
	}
}

func TestBrowserEnterPrintsSelection(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)

	done := press(typeText(m, "greet"), "enter").(browserModel)
	if !done.quitting || done.result.Action != ActionSelect || done.result.Entry == nil || done.result.Entry.Name != "greet" {
		t.Fatalf("enter from the search should select the top match, got %+v", done.result)
	}
	if done.View() != "" {
		t.Error("view should clear on exit")
	}

	fromList := press(m, "down", "down", "enter").(browserModel)
	if fromList.result.Entry == nil || fromList.result.Entry.Name != "greet" {
		t.Fatalf("↓ enters the list, ↓ moves, enter selects; got %+v", fromList.result)
	}
}

func TestBrowserFocusSwitching(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)

	for _, k := range []string{"esc", "down"} {
		inList := press(m, k).(browserModel)
		if inList.focus != focusList || inList.search.Focused() {
			t.Fatalf("%s should move focus to the list", k)
		}
		if inList.list.Index() != 0 {
			t.Errorf("%s should keep the first item selected, got %d", k, inList.list.Index())
		}
	}

	inList := press(m, "esc").(browserModel)
	moved := press(inList, "j", "j").(browserModel)
	if moved.list.Index() != 2 || moved.preview.Entry().Name != "tool" {
		t.Fatalf("j should move through the list, index %d", moved.list.Index())
	}
	if moved.search.Value() != "" {
		t.Errorf("list keys should not type into the search, got %q", moved.search.Value())
	}

	for _, k := range []string{"/", "esc"} {
		back := press(inList, k).(browserModel)
		if back.focus != focusSearch || back.quitting {
			t.Errorf("%s in the list should refocus the search", k)
		}
	}
	top := press(inList, "up").(browserModel)
	if top.focus != focusSearch {
		t.Error("↑ on the first item should refocus the search")
	}
	notTop := press(inList, "down", "up").(browserModel)
	if notTop.focus != focusList || notTop.list.Index() != 0 {
		t.Error("↑ below the first item should just move up")
	}

	// ctrl+n / ctrl+p move the selection without leaving the search.
	nav := press(m, "ctrl+n").(browserModel)
	if nav.focus != focusSearch || nav.list.Index() != 1 {
		t.Errorf("ctrl+n should move the selection while searching, index %d", nav.list.Index())
	}
}

func TestBrowserQuit(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	for name, model := range map[string]tea.Model{
		"ctrl+c in search": press(m, "ctrl+c"),
		"ctrl+c in list":   press(m, "esc", "ctrl+c"),
		"q in list":        press(m, "esc", "q"),
	} {
		done := model.(browserModel)
		if !done.quitting || done.result.Action != ActionNone {
			t.Errorf("%s: expected quit without selection, got %+v", name, done.result)
		}
	}
}

func TestBrowserPrefilledFilter(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{Filter: "gre"}), 120, 30)
	if m.search.Value() != "gre" || m.focus != focusSearch {
		t.Fatalf("search should be pre-filled and focused, got %q", m.search.Value())
	}
	if got := len(m.list.VisibleItems()); got != 1 {
		t.Fatalf("expected 1 match for 'gre', got %d", got)
	}
	if e := m.preview.Entry(); e == nil || e.Name != "greet" {
		t.Fatalf("preview should follow the filtered selection, got %+v", e)
	}
	// Keep typing to refine; backspace widens.
	if got := len(typeText(m, "z").(browserModel).list.VisibleItems()); got != 0 {
		t.Errorf("expected no matches for 'grez', got %d", got)
	}
}

func TestBrowserPassesFilterToPreview(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{Filter: "gre"}), 120, 30)
	if m.preview.query != "gre" {
		t.Fatalf("preview query = %q, want the CLI filter", m.preview.query)
	}
	cleared := press(m, "backspace", "backspace", "backspace").(browserModel)
	if cleared.preview.query != "" {
		t.Fatalf("clearing the search should clear the preview query, got %q", cleared.preview.query)
	}
	typing := typeText(cleared, "as").(browserModel)
	if typing.preview.query != "as" {
		t.Fatalf("typing should update the preview query, got %q", typing.preview.query)
	}
}

func TestBrowserTypeLine(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	searchRow, typeRow := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "❯") && searchRow < 0 {
			searchRow = i
		}
		if strings.Contains(line, "all  aliases  funcs  bin") && typeRow < 0 {
			typeRow = i
		}
	}
	if searchRow < 0 || typeRow != searchRow+1 {
		t.Fatalf("type line should sit directly under the search (search %d, type %d):\n%s", searchRow, typeRow, strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[typeRow], "3/3") {
		t.Errorf("type line should show the match count, got %q", lines[typeRow])
	}
	if got := len(typeText(m, "gre").(browserModel).list.VisibleItems()); got != 1 {
		t.Fatal("setup")
	}
	if row := ansi.Strip(typeText(m, "gre").(browserModel).typeLine(80)); !strings.Contains(row, "1/3") {
		t.Errorf("count should follow the search, got %q", row)
	}
}

func TestBrowserTypeLineEmphasizesCurrentType(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	theme := testTheme()
	theme.Primary = lipgloss.Color("2")
	theme.Muted = lipgloss.Color("8")
	m := sized(newBrowserModel(fixtureEntries(t), theme, BrowserOptions{}), 120, 30)

	activeStyle := regexp.MustCompile(`\x1b\[1;32m([a-z]+)`)
	for _, want := range []string{"all", "aliases", "funcs", "bin"} {
		got := activeStyle.FindAllStringSubmatch(m.typeLine(80), -1)
		if len(got) != 1 || got[0][1] != want {
			t.Errorf("expected only %q emphasized, got %v", want, got)
		}
		m = press(m, "tab").(browserModel)
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
	}
	back := press(model, "shift+tab").(browserModel)
	if back.typeFilter != FilterExecutable {
		t.Errorf("shift+tab should cycle backwards, got %v", back.typeFilter)
	}

	// The search is kept and re-applied when the type changes.
	searched := press(typeText(m, "s"), "tab").(browserModel)
	if searched.search.Value() != "s" || len(searched.list.VisibleItems()) != 1 || searched.preview.Entry().Name != "asdf" {
		t.Errorf("type change should keep the search, value %q items %d", searched.search.Value(), len(searched.list.VisibleItems()))
	}
}

func TestBrowserCopy(t *testing.T) {
	var copied []string
	copyFn := func(s string) error { copied = append(copied, s); return nil }
	entries := fixtureEntries(t)
	m := sized(newBrowserModel(entries, testTheme(), BrowserOptions{Copy: copyFn}), 120, 30)

	inList := press(m, "esc", "y", "Y", "c").(browserModel)
	want := []string{`alias asdf="cowsay 'hi'"`, entries[0].File + ":40", "asdf"}
	if strings.Join(copied, "|") != strings.Join(want, "|") {
		t.Fatalf("copied %q, want %q", copied, want)
	}
	if inList.quitting || !strings.Contains(inList.status, "copied name") {
		t.Errorf("copy should stay open with a status, got quitting=%v status=%q", inList.quitting, inList.status)
	}

	// ctrl+y copies while still typing in the search.
	copied = nil
	searching := press(m, "ctrl+y").(browserModel)
	if len(copied) != 1 || copied[0] != want[0] || searching.focus != focusSearch {
		t.Errorf("ctrl+y from the search: copied %q", copied)
	}

	// Executables copy their path.
	copied = nil
	press(m, "esc", "j", "j", "y")
	if len(copied) != 1 || copied[0] != entries[2].File {
		t.Errorf("executable copy = %q", copied)
	}

	failing := sized(newBrowserModel(entries, testTheme(), BrowserOptions{Copy: func(string) error { return errors.New("no clipboard") }}), 120, 30)
	if got := press(failing, "esc", "y").(browserModel).status; !strings.Contains(got, "no clipboard") {
		t.Errorf("status = %q", got)
	}
}

func TestBrowserOpen(t *testing.T) {
	m := sized(newBrowserModel(fixtureEntries(t), testTheme(), BrowserOptions{}), 120, 30)
	for name, model := range map[string]tea.Model{
		"o in list":        press(m, "esc", "o"),
		"ctrl+o in search": press(m, "ctrl+o"),
	} {
		done := model.(browserModel)
		if !done.quitting || done.result.Action != ActionOpen || done.result.Entry.Name != "asdf" {
			t.Fatalf("%s: got %+v", name, done.result)
		}
	}

	exe := press(m, "esc", "j", "j", "o").(browserModel)
	if exe.quitting || !strings.Contains(exe.status, "executable") {
		t.Fatalf("o on an executable should stay open with a status, got quitting=%v %q", exe.quitting, exe.status)
	}
}

func TestBrowserResponsiveHelp(t *testing.T) {
	entries := fixtureEntries(t)
	for width, want := range map[int]int{30: 2, 50: 3, 70: 4} {
		m := sized(newBrowserModel(entries, testTheme(), BrowserOptions{}), width, 30)
		if got := len(m.shortHelpKeys()); got != want {
			t.Errorf("width %d: %d short help keys, want %d", width, got, want)
		}
	}
	wide := sized(newBrowserModel(entries, testTheme(), BrowserOptions{}), 120, 30)
	searchHelp := ansi.Strip(wide.footer())
	for _, want := range []string{"browse list", "ctrl+o", "ctrl+y"} {
		if !strings.Contains(searchHelp, want) {
			t.Errorf("search help should mention %q: %q", want, searchHelp)
		}
	}
	listHelp := ansi.Strip(press(wide, "esc").(browserModel).footer())
	for _, want := range []string{"/ search", "o open", "? more"} {
		if !strings.Contains(listHelp, want) {
			t.Errorf("list help should mention %q: %q", want, listHelp)
		}
	}

	full := press(wide, "esc", "?").(browserModel)
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

	searching := press(m, "shift+down", "shift+down").(browserModel)
	if searching.preview.YOffset() != start+2 || searching.focus != focusSearch {
		t.Errorf("shift+↓ should scroll the preview from the search: %d -> %d", start, searching.preview.YOffset())
	}

	inList := press(m, "esc", "J", "J", "K").(browserModel)
	if inList.preview.YOffset() != start+1 {
		t.Errorf("J/K should scroll the preview in the list: got %d", inList.preview.YOffset())
	}
	if inList.list.Index() != 0 {
		t.Error("preview scrolling should not move the list selection")
	}

	// In the search, ctrl+u is the text input's delete-to-start, not a scroll.
	edited := press(typeText(m, "gre"), "ctrl+u").(browserModel)
	if edited.search.Value() != "" {
		t.Errorf("ctrl+u should clear the search, got %q", edited.search.Value())
	}
}
