package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/imdevan/witch/internal/domain"
)

// BrowserAction is what the user chose to do with the selected entry.
type BrowserAction int

const (
	// ActionNone means the user quit without choosing.
	ActionNone BrowserAction = iota
	// ActionPrint prints the selection (enter).
	ActionPrint
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
	Filter  string // pre-filled fuzzy filter (e.g. the CLI argument)
	Spacing string // list spacing: compact, tight, space
	// Copy writes text to the clipboard. Nil disables the copy keys.
	Copy func(string) error
	// MaxHeight caps the inline UI height (default 40 lines).
	MaxHeight int
}

type browserKeys struct {
	Select     key.Binding
	Back       key.Binding
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
}

func newBrowserKeys() browserKeys {
	return browserKeys{
		Select:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:       key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Open:       key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in editor")),
		CopyDef:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy definition")),
		CopyPath:   key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy path:line")),
		CopyName:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy name")),
		NextType:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next type")),
		PrevType:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev type")),
		ScrollDown: key.NewBinding(key.WithKeys("J", "shift+down"), key.WithHelp("J", "scroll preview")),
		ScrollUp:   key.NewBinding(key.WithKeys("K", "shift+up"), key.WithHelp("K", "scroll preview up")),
		HalfDown:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "preview ½ page down")),
		HalfUp:     key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "preview ½ page up")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Quit:       key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
		ForceQuit:  key.NewBinding(key.WithKeys("ctrl+c")),
	}
}

type browserModel struct {
	entries    []domain.Entry
	list       list.Model
	preview    Preview
	help       help.Model
	keys       browserKeys
	theme      Theme
	responsive *ResponsiveManager
	layout     SplitLayout
	typeFilter TypeFilter
	title      string
	maxHeight  int
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
		title:      opts.Title,
		maxHeight:  opts.MaxHeight,
		copy:       opts.Copy,
	}
	if m.title == "" {
		m.title = "witch"
	}
	if m.maxHeight <= 0 {
		m.maxHeight = 40
	}

	m.list = NewListModel(entryItems(entries, FilterAll), newEntryDelegate(theme, opts.Spacing), 0, 0, theme)
	m.list.SetShowHelp(false)
	m.list.SetShowStatusBar(true)
	m.list.SetFilteringEnabled(true)
	m.list.Title = m.listTitle()
	if opts.Filter != "" {
		m.list.SetFilterText(opts.Filter)
	}

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

func (m browserModel) Init() tea.Cmd { return nil }

func (m browserModel) listTitle() string {
	if m.typeFilter == FilterAll {
		return m.title
	}
	return m.title + " · " + m.typeFilter.Label()
}

func (m browserModel) split() bool { return ShouldSplit(m.responsive.Breakpoint()) }

// resize lays out the panes for a terminal of the given size.
func (m *browserModel) resize(width, height int) {
	m.width, m.height = width, height
	m.responsive.SetWidth(width)
	m.help.Width = width

	// Inline: leave the last terminal row for the cursor, and stay compact.
	total := max(min(height-1, m.maxHeight), 8)
	m.layout.Width = width
	m.layout.Height = max(total-lipgloss.Height(m.footer()), 5)

	if m.split() {
		lw, rw, h := m.layout.ContentSizes()
		m.list.SetSize(lw, h)
		m.preview.SetSize(rw, h)
		m.previewMode = false
		return
	}
	w, h := m.layout.SingleContentSize()
	m.list.SetSize(w, h)
	m.preview.SetSize(w, h)
}

// syncPreview points the preview at the selected list item and passes it
// the current filter text to highlight.
func (m *browserModel) syncPreview() {
	if item, ok := m.list.SelectedItem().(EntryItem); ok {
		e := item.Entry
		m.preview.SetEntry(&e)
	} else {
		m.preview.SetEntry(nil)
	}
	if m.list.FilterState() == list.Unfiltered {
		m.preview.SetQuery("")
	} else {
		m.preview.SetQuery(m.list.FilterValue())
	}
}

func (m browserModel) selected() *domain.Entry {
	if item, ok := m.list.SelectedItem().(EntryItem); ok {
		e := item.Entry
		return &e
	}
	return nil
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
			return m.cycleType(delta)
		}
		if m.list.SettingFilter() {
			break
		}
		if m.previewMode {
			return m.updatePreviewMode(msg)
		}
		if model, cmd, handled := m.handleKey(msg); handled {
			return model, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncPreview()
	return m, cmd
}

// handleKey handles browsing keys; handled=false passes the key to the list.
func (m browserModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if msg.String() == "esc" && m.list.FilterState() == list.FilterApplied {
			return m, nil, false // the list clears its filter
		}
		model, cmd := m.quit(BrowserResult{})
		return model, cmd, true
	case key.Matches(msg, m.keys.Select):
		if m.selected() == nil {
			return m, nil, true
		}
		if !m.split() {
			m.previewMode = true
			return m, nil, true
		}
		model, cmd := m.quit(BrowserResult{Entry: m.selected(), Action: ActionPrint})
		return model, cmd, true
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.resize(m.width, m.height)
		return m, nil, true
	case key.Matches(msg, m.keys.ScrollDown):
		m.preview.ScrollDown(1)
		return m, nil, true
	case key.Matches(msg, m.keys.ScrollUp):
		m.preview.ScrollDown(-1)
		return m, nil, true
	case key.Matches(msg, m.keys.HalfDown):
		m.preview.ScrollDown(m.preview.HalfPage())
		return m, nil, true
	case key.Matches(msg, m.keys.HalfUp):
		m.preview.ScrollDown(-m.preview.HalfPage())
		return m, nil, true
	}
	if model, cmd, handled := m.handleAction(msg); handled {
		return model, cmd, true
	}
	return m, nil, false
}

// updatePreviewMode handles keys in the narrow-screen preview.
func (m browserModel) updatePreviewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.previewMode = false
		return m, nil
	case msg.String() == "q":
		return m.quit(BrowserResult{})
	case key.Matches(msg, m.keys.Select):
		return m.quit(BrowserResult{Entry: m.selected(), Action: ActionPrint})
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

// handleAction handles open and copy keys, shared by both modes.
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

func (m browserModel) cycleType(delta int) (tea.Model, tea.Cmd) {
	m.typeFilter = m.typeFilter.Next(delta)
	m.list.Title = m.listTitle()
	cmd := m.list.SetItems(entryItems(m.entries, m.typeFilter))
	m.list.ResetSelected()
	m.syncPreview()
	return m, cmd
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
	all := []key.Binding{enter, m.list.KeyMap.Filter, m.keys.NextType, m.keys.Open, m.keys.CopyDef, m.keys.Quit, m.keys.Help}
	if m.previewMode {
		all = []key.Binding{enter, m.keys.Back, m.keys.Open, m.keys.CopyDef, m.keys.Quit, m.keys.Help}
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
	if keep >= len(all) {
		return all
	}
	// Always keep the help toggle so the rest stays discoverable.
	return append(all[:keep-1:keep-1], m.keys.Help)
}

func (m browserModel) fullHelpKeys() [][]key.Binding {
	return [][]key.Binding{
		{m.list.KeyMap.CursorUp, m.list.KeyMap.CursorDown, m.list.KeyMap.Filter, m.list.KeyMap.ClearFilter, m.keys.NextType, m.keys.PrevType},
		{m.keys.Select, m.keys.Open, m.keys.CopyDef, m.keys.CopyPath, m.keys.CopyName},
		{m.keys.ScrollDown, m.keys.ScrollUp, m.keys.HalfDown, m.keys.HalfUp, m.keys.Quit},
	}
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
		body = m.layout.Render(m.list.View(), m.preview.View())
	case m.previewMode:
		body = m.layout.RenderSingle(m.preview.View())
	default:
		body = m.layout.RenderSingle(m.list.View())
	}
	return body + "\n" + m.footer()
}
