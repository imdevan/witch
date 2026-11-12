package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/imdevan/witch/internal/domain"
)

// EntryPickerOptions configures the interactive entry picker.
type EntryPickerOptions struct {
	Title   string
	Filter  string // pre-filled filter text
	Spacing string // list spacing: compact, tight, space
}

// EntryItem adapts a domain.Entry to a bubbles list item.
type EntryItem struct {
	Entry domain.Entry
}

// Title is the entry name.
func (i EntryItem) Title() string { return i.Entry.Name }

// Description shows the type plus the most useful detail available.
func (i EntryItem) Description() string {
	parts := []string{i.Entry.TypeLabel()}
	if i.Entry.Shadowed {
		parts = append(parts, "shadowed")
	}
	detail := i.Entry.Description
	if detail == "" && i.Entry.Type == domain.TypeAlias {
		detail = i.Entry.Value
	}
	if detail == "" {
		detail = i.Entry.Location()
	}
	if i.Entry.Target != "" {
		detail += " → " + i.Entry.Target
	}
	return strings.Join(parts, " · ") + " · " + firstLine(detail)
}

// FilterValue filters on the entry name.
func (i EntryItem) FilterValue() string { return i.Entry.Name }

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

type entryPickerModel struct {
	list       list.Model
	theme      Theme
	responsive *ResponsiveManager
	selected   *domain.Entry
	quitting   bool
}

// RunEntryPicker shows an inline, filterable list of entries and returns the
// one the user picked, or nil if they quit without choosing.
func RunEntryPicker(entries []domain.Entry, theme Theme, opts EntryPickerOptions, programOpts ...tea.ProgramOption) (*domain.Entry, error) {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = EntryItem{Entry: e}
	}

	delegate := NewListDelegate(theme, ListDelegateOptions{Spacing: opts.Spacing})
	model := entryPickerModel{
		list:       NewListModel(items, delegate, 80, 20, theme),
		theme:      theme,
		responsive: NewResponsiveManager(80),
	}
	model.list.Title = opts.Title
	if model.list.Title == "" {
		model.list.Title = "witch"
	}
	model.list.SetShowStatusBar(true)
	model.list.SetFilteringEnabled(true)
	model.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))}
	}
	if opts.Filter != "" {
		model.list.SetFilterText(opts.Filter)
	}

	result, err := tea.NewProgram(model, programOpts...).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run interactive list: %w", err)
	}
	if final, ok := result.(entryPickerModel); ok {
		return final.selected, nil
	}
	return nil, nil
}

func (m entryPickerModel) Init() tea.Cmd { return nil }

func (m entryPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.responsive.SetWidth(msg.Width)
		width, height := m.responsive.GetListDimensions(msg.Width, msg.Height)
		m.list.SetSize(width, min(height, msg.Height-2))
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "q", "esc":
				if msg.String() == "esc" && m.list.FilterState() == list.FilterApplied {
					break // let the list clear the filter
				}
				m.quitting = true
				return m, tea.Quit
			case "enter":
				if item, ok := m.list.SelectedItem().(EntryItem); ok {
					entry := item.Entry
					m.selected = &entry
				}
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m entryPickerModel) View() string {
	if m.quitting {
		// Clear the inline UI so only the printed result remains.
		return ""
	}
	return m.responsive.AdaptiveFrameStyle(m.theme).Render(m.list.View())
}
