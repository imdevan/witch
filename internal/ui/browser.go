package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/imdevan/witch/internal/domain"
)

// BrowserAction is what the user chose to do with the selected entry.
type BrowserAction int

const (
	// ActionNone means the user quit without choosing.
	ActionNone BrowserAction = iota
	// ActionSelect is enter: the CLI executes or prints the selection.
	ActionSelect
	// ActionOpen opens the selection's defining file in an editor (o).
	ActionOpen
)

// BrowserResult is returned when the browser exits.
type BrowserResult struct {
	Entry  *domain.Entry
	Action BrowserAction
}

// BrowserOptions configures the browser.
type BrowserOptions struct {
	Title   string
	Filter  string // pre-filled search (e.g. the CLI argument)
	Spacing string // list spacing: compact, tight, space
	// Copy writes text to the clipboard. Nil disables the copy keys.
	Copy func(string) error
	// MaxHeight caps the inline UI height (default 40 lines). Ignored when
	// Fullscreen is set.
	MaxHeight int
	// Fullscreen renders on the alternate screen using the whole terminal
	// pane instead of inline below the prompt.
	Fullscreen bool
}

// browserFocus is where keystrokes go.
type browserFocus int

const (
	// focusSearch: typing edits the search, which filters the list.
	focusSearch browserFocus = iota
	// focusList: keys navigate the list and run single-letter actions.
	focusList
)

type browserKeys struct {
	Select     key.Binding
	Back       key.Binding
	ToList     key.Binding
	ToSearch   key.Binding
	SearchNav  key.Binding
	Open       key.Binding
	CopyDef    key.Binding
	CopyPath   key.Binding
	CopyName   key.Binding
	NextType   key.Binding
	PrevType   key.Binding
	ScrollDown key.Binding
	ScrollUp   key.Binding
	HalfDown   key.Binding
	HalfUp     key.Binding
	Help       key.Binding
	Quit       key.Binding
	ForceQuit  key.Binding
	// Chorded variants that also work while typing in the search.
	OpenChord key.Binding
	CopyChord key.Binding
}

func newBrowserKeys() browserKeys {
	return browserKeys{
		Select:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:       key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		ToList:     key.NewBinding(key.WithKeys("esc", "down"), key.WithHelp("esc/↓", "browse list")),
		ToSearch:   key.NewBinding(key.WithKeys("/", "esc"), key.WithHelp("/", "search")),
		SearchNav:  key.NewBinding(key.WithKeys("ctrl+n", "ctrl+p"), key.WithHelp("ctrl+n/p", "move selection")),
		Open:       key.NewBinding(key.WithKeys("o", "ctrl+o"), key.WithHelp("o", "open in editor")),
		CopyDef:    key.NewBinding(key.WithKeys("y", "ctrl+y"), key.WithHelp("y", "copy definition")),
		CopyPath:   key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy path:line")),
		CopyName:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy name")),
		NextType:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next type")),
		PrevType:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev type")),
		ScrollDown: key.NewBinding(key.WithKeys("J", "shift+down"), key.WithHelp("J/shift+↓", "scroll preview")),
		ScrollUp:   key.NewBinding(key.WithKeys("K", "shift+up"), key.WithHelp("K/shift+↑", "scroll preview up")),
		HalfDown:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "preview ½ page down")),
		HalfUp:     key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "preview ½ page up")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		ForceQuit:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		OpenChord:  key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "open in editor")),
		CopyChord:  key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "copy definition")),
	}
}

type browserModel struct {
	entries    []domain.Entry
	search     textinput.Model
	list       list.Model
	preview    Preview
	help       help.Model
	keys       browserKeys
	theme      Theme
	responsive *ResponsiveManager
	layout     SplitLayout
	typeFilter TypeFilter
	focus      browserFocus
	maxHeight  int
	fullscreen bool
	width      int
	height     int
	// previewMode is the narrow-screen preview opened with enter.
	previewMode bool
	status      string
	copy        func(string) error
	result      BrowserResult
	quitting    bool
}

// RunBrowser shows the inline split list/preview browser and returns what
// the user selected.
func RunBrowser(entries []domain.Entry, theme Theme, opts BrowserOptions, programOpts ...tea.ProgramOption) (BrowserResult, error) {
	if opts.Fullscreen {
		programOpts = append(programOpts, tea.WithAltScreen())
	}
	final, err := tea.NewProgram(newBrowserModel(entries, theme, opts), programOpts...).Run()
	if err != nil {
		return BrowserResult{}, fmt.Errorf("failed to run interactive list: %w", err)
	}
	if m, ok := final.(browserModel); ok {
		return m.result, nil
	}
	return BrowserResult{}, nil
}

func newBrowserModel(entries []domain.Entry, theme Theme, opts BrowserOptions) browserModel {
	m := browserModel{
		entries:    entries,
		preview:    NewPreview(theme),
		help:       help.New(),
		keys:       newBrowserKeys(),
		theme:      theme,
		responsive: NewResponsiveManager(80),
		layout:     NewSplitLayout(theme),
		maxHeight:  opts.MaxHeight,
		fullscreen: opts.Fullscreen,
		copy:       opts.Copy,
	}
	if m.maxHeight <= 0 {
		m.maxHeight = 40
	}

	m.search = textinput.New()
	m.search.Prompt = "❯ "
	m.search.Placeholder = "type to search"
	m.search.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary)
	m.search.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	m.search.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)
	m.search.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Secondary)
	m.search.SetValue(opts.Filter)
	m.search.CursorEnd()
	m.search.Focus()

	// The search input above replaces the list's own title, filter prompt,
	// and status bar; the list only shows matching items.
	m.list = NewListModel(entryItems(entries, FilterAll), newEntryDelegate(theme, opts.Spacing), 0, 0, theme)
	m.list.SetShowHelp(false)
	m.list.SetShowTitle(false)
	m.list.SetShowFilter(false)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.KeyMap.Filter.SetEnabled(false)
	m.applySearch()

	m.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(theme.Secondary)
	m.help.Styles.FullKey = m.help.Styles.ShortKey
	m.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(theme.Muted)
	m.help.Styles.FullDesc = m.help.Styles.ShortDesc
	m.help.Styles.ShortSeparator = m.help.Styles.ShortDesc
	m.help.Styles.FullSeparator = m.help.Styles.ShortDesc
	m.help.Styles.Ellipsis = m.help.Styles.ShortDesc

	m.resize(80, 24)
	m.syncPreview()
	return m
}

func (m browserModel) Init() tea.Cmd { return textinput.Blink }

func (m browserModel) split() bool { return ShouldSplit(m.responsive.Breakpoint()) }

// headerHeight is the search line plus the type line above the list.
const headerHeight = 2

// resize lays out the panes for a terminal of the given size.
func (m *browserModel) resize(width, height int) {
	m.width, m.height = width, height
	m.responsive.SetWidth(width)
	m.help.Width = width

	total := height
	if !m.fullscreen {
		// Inline: leave the last row for the cursor and stay compact.
		total = min(height-1, m.maxHeight)
	}
	total = max(total, 8)
	m.layout.Width = width
	m.layout.Height = max(total-lipgloss.Height(m.footer()), 5)

	var listWidth, h int
	if m.split() {
		var rw int
		listWidth, rw, h = m.layout.ContentSizes()
		m.preview.SetSize(rw, h)
		m.previewMode = false
	} else {
		listWidth, h = m.layout.SingleContentSize()
		m.preview.SetSize(listWidth, h)
	}
	m.list.SetSize(listWidth, max(h-headerHeight, 1))
	m.search.Width = max(listWidth-lipgloss.Width(m.search.Prompt)-1, 1)
}

// applySearch filters the list with the current search text. Filtering is
// synchronous so the selection and preview update on every keystroke.
func (m *browserModel) applySearch() {
	if q := m.search.Value(); strings.TrimSpace(q) != "" {
		m.list.SetFilterText(q)
	} else {
		m.list.ResetFilter()
	}
	m.syncPreview()
}

// syncPreview points the preview at the selected list item and passes it
// the current search to highlight.
func (m *browserModel) syncPreview() {
	if item, ok := m.list.SelectedItem().(EntryItem); ok {
		e := item.Entry
		m.preview.SetEntry(&e)
	} else {
		m.preview.SetEntry(nil)
	}
	m.preview.SetQuery(m.search.Value())
}

func (m browserModel) selected() *domain.Entry {
	if item, ok := m.list.SelectedItem().(EntryItem); ok {
		e := item.Entry
		return &e
	}
	return nil
}

func (m *browserModel) setFocus(f browserFocus) {
	m.focus = f
	if f == focusSearch {
		m.search.Focus()
		m.search.PromptStyle = lipgloss.NewStyle().Foreground(m.theme.Secondary)
	} else {
		m.search.Blur()
		m.search.PromptStyle = lipgloss.NewStyle().Foreground(m.theme.Muted)
	}
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		m.status = ""
		if key.Matches(msg, m.keys.ForceQuit) {
			return m.quit(BrowserResult{})
		}
		if key.Matches(msg, m.keys.NextType, m.keys.PrevType) {
			delta := 1
			if key.Matches(msg, m.keys.PrevType) {
				delta = -1
			}
			return m.cycleType(delta), nil
		}
		if m.previewMode {
			return m.updatePreviewMode(msg)
		}
		if model, cmd, handled := m.handleSharedKey(msg); handled {
			return model, cmd
		}
		if m.focus == focusSearch {
			return m.updateSearch(msg)
		}
		return m.updateList(msg)
	}

	// Cursor blink and other non-key messages.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// handleSharedKey handles keys that behave the same in search and list focus.
func (m browserModel) handleSharedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Select):
		if m.selected() == nil {
			return m, nil, true
		}
		if !m.split() {
			m.previewMode = true
			return m, nil, true
		}
		model, cmd := m.quit(BrowserResult{Entry: m.selected(), Action: ActionSelect})
		return model, cmd, true
	case msg.String() == "shift+down":
		m.preview.ScrollDown(1)
		return m, nil, true
	case msg.String() == "shift+up":
		m.preview.ScrollDown(-1)
		return m, nil, true
	case key.Matches(msg, m.keys.OpenChord, m.keys.CopyChord):
		model, cmd, _ := m.handleAction(msg)
		return model, cmd, true
	}
	return m, nil, false
}

// updateSearch handles keys while the search input has focus: everything
// typed edits the search; esc or ↓ moves focus to the list.
func (m browserModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.ToList):
		if len(m.list.VisibleItems()) > 0 {
			m.setFocus(focusList)
		}
		return m, nil
	case msg.String() == "ctrl+n":
		m.list.CursorDown()
		m.syncPreview()
		return m, nil
	case msg.String() == "ctrl+p":
		m.list.CursorUp()
		m.syncPreview()
		return m, nil
	case msg.String() == "up":
		return m, nil
	}
	before := m.search.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if m.search.Value() != before {
		m.applySearch()
	}
	return m, cmd
}

// updateList handles keys while the list has focus: navigation plus
// single-letter actions; / or esc returns to the search.
func (m browserModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.ToSearch):
		m.setFocus(focusSearch)
		return m, nil
	case key.Matches(msg, m.list.KeyMap.CursorUp) && m.list.Index() == 0:
		// ↑ past the first item goes back to the search.
		m.setFocus(focusSearch)
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		return m.quit(BrowserResult{})
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.resize(m.width, m.height)
		return m, nil
	case key.Matches(msg, m.keys.ScrollDown):
		m.preview.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.ScrollUp):
		m.preview.ScrollDown(-1)
		return m, nil
	case key.Matches(msg, m.keys.HalfDown):
		m.preview.ScrollDown(m.preview.HalfPage())
		return m, nil
	case key.Matches(msg, m.keys.HalfUp):
		m.preview.ScrollDown(-m.preview.HalfPage())
		return m, nil
	}
	if model, cmd, handled := m.handleAction(msg); handled {
		return model, cmd
	}
	// Navigation (↑/↓, j/k, pgup/pgdn, g/G) is handled by the list. Keys
	// that would change the list's own filter state are not forwarded.
	if key.Matches(msg, m.list.KeyMap.ClearFilter, m.list.KeyMap.Quit, m.list.KeyMap.ShowFullHelp, m.list.KeyMap.CloseFullHelp) {
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncPreview()
	return m, cmd
}

// updatePreviewMode handles keys in the narrow-screen preview.
func (m browserModel) updatePreviewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.previewMode = false
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		return m.quit(BrowserResult{})
	case key.Matches(msg, m.keys.Select):
		return m.quit(BrowserResult{Entry: m.selected(), Action: ActionSelect})
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.resize(m.width, m.height)
		return m, nil
	}
	if model, cmd, handled := m.handleAction(msg); handled {
		return model, cmd
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

// handleAction handles open and copy keys.
func (m browserModel) handleAction(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	e := m.selected()
	switch {
	case key.Matches(msg, m.keys.Open):
		if e == nil {
			return m, nil, true
		}
		if e.Type == domain.TypeExecutable || e.Line <= 0 {
			m.status = fmt.Sprintf("%s is an executable; it has no definition to open", e.Name)
			return m, nil, true
		}
		model, cmd := m.quit(BrowserResult{Entry: e, Action: ActionOpen})
		return model, cmd, true
	case key.Matches(msg, m.keys.CopyDef, m.keys.CopyPath, m.keys.CopyName):
		if e == nil {
			return m, nil, true
		}
		what, text := "definition", copyDefinition(*e)
		if key.Matches(msg, m.keys.CopyPath) {
			what, text = "path", e.Location()
		} else if key.Matches(msg, m.keys.CopyName) {
			what, text = "name", e.Name
		}
		m.status = m.copyText(what, text)
		return m, nil, true
	}
	return m, nil, false
}

func copyDefinition(e domain.Entry) string {
	if e.Type == domain.TypeExecutable || e.Definition == "" {
		return e.File
	}
	return e.Definition
}

func (m browserModel) copyText(what, text string) string {
	if m.copy == nil {
		return "clipboard unavailable"
	}
	if err := m.copy(text); err != nil {
		return "copy failed: " + err.Error()
	}
	return fmt.Sprintf("copied %s: %s", what, firstLine(text))
}

func (m browserModel) cycleType(delta int) browserModel {
	m.typeFilter = m.typeFilter.Next(delta)
	m.list.SetItems(entryItems(m.entries, m.typeFilter))
	m.list.ResetSelected()
	m.applySearch()
	if len(m.list.VisibleItems()) == 0 && m.focus == focusList {
		m.setFocus(focusSearch)
	}
	return m
}

func (m browserModel) quit(result BrowserResult) (tea.Model, tea.Cmd) {
	m.result = result
	m.quitting = true
	return m, tea.Quit
}

// shortHelpKeys returns the bindings for the one-line help, trimmed by
// breakpoint so narrow terminals show only the essentials.
func (m browserModel) shortHelpKeys() []key.Binding {
	enter := m.keys.Select
	if !m.split() && !m.previewMode {
		enter = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "preview"))
	}
	var all []key.Binding
	switch {
	case m.previewMode:
		all = []key.Binding{enter, m.keys.Back, m.keys.Open, m.keys.CopyDef, m.keys.Quit, m.keys.Help}
	case m.focus == focusSearch:
		all = []key.Binding{enter, m.keys.ToList, m.keys.NextType, m.keys.OpenChord, m.keys.CopyChord, m.keys.ForceQuit}
	default:
		all = []key.Binding{enter, m.keys.ToSearch, m.keys.NextType, m.keys.Open, m.keys.CopyDef, m.keys.Quit, m.keys.Help}
	}
	var keep int
	switch m.responsive.Breakpoint() {
	case BreakpointXS:
		keep = 2
	case BreakpointSM:
		keep = 3
	case BreakpointMD:
		keep = 4
	default:
		keep = len(all)
	}
	return all[:min(keep, len(all))]
}

func (m browserModel) fullHelpKeys() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.ToList, m.keys.ToSearch, m.list.KeyMap.CursorUp, m.list.KeyMap.CursorDown, m.keys.SearchNav, m.keys.NextType, m.keys.PrevType},
		{m.keys.Select, m.keys.Open, m.keys.CopyDef, m.keys.CopyPath, m.keys.CopyName},
		{m.keys.ScrollDown, m.keys.ScrollUp, m.keys.HalfDown, m.keys.HalfUp, m.keys.Quit},
	}
}

// typeLine renders "all aliases funcs bin" with the active type
// emphasized, plus the match count on the right.
func (m browserModel) typeLine(width int) string {
	muted := lipgloss.NewStyle().Foreground(m.theme.Muted)
	active := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	var parts []string
	for f := FilterAll; f < typeFilterCount; f++ {
		if f == m.typeFilter {
			parts = append(parts, active.Render(f.ShortLabel()))
		} else {
			parts = append(parts, muted.Render(f.ShortLabel()))
		}
	}
	left := "  " + strings.Join(parts, "  ")
	count := muted.Render(fmt.Sprintf("%d/%d", len(m.list.VisibleItems()), len(m.list.Items())))
	gap := width - lipgloss.Width(left) - lipgloss.Width(count)
	if gap < 1 {
		return lipgloss.NewStyle().MaxWidth(width).Render(left)
	}
	return left + strings.Repeat(" ", gap) + count
}

func (m browserModel) listPane() string {
	width := m.list.Width()
	return m.search.View() + "\n" + m.typeLine(width) + "\n" + m.list.View()
}

func (m browserModel) footer() string {
	status := lipgloss.NewStyle().Foreground(m.theme.Secondary).Render(m.status)
	var helpView string
	if m.help.ShowAll {
		helpView = m.help.FullHelpView(m.fullHelpKeys())
	} else {
		helpView = m.help.ShortHelpView(m.shortHelpKeys())
	}
	return status + "\n" + lipgloss.NewStyle().PaddingLeft(1).Render(helpView)
}

func (m browserModel) View() string {
	if m.quitting {
		// Clear the inline UI so only the printed result remains.
		return ""
	}
	var body string
	switch {
	case m.split():
		body = m.layout.Render(m.listPane(), m.preview.View())
	case m.previewMode:
		body = m.layout.RenderSingle(m.preview.View())
	default:
		body = m.layout.RenderSingle(m.listPane())
	}
	return body + "\n" + m.footer()
}
