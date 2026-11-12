package ui

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/utils"
)

// EntryItem adapts a domain.Entry to a bubbles list item.
type EntryItem struct {
	Entry domain.Entry
}

// Title is the entry name.
func (i EntryItem) Title() string { return i.Entry.Name }

// Description is the type followed by the most useful detail available.
func (i EntryItem) Description() string {
	return i.Entry.TypeLabel() + " " + i.Detail()
}

// Detail is the description text shown after the type badge.
func (i EntryItem) Detail() string {
	var parts []string
	if i.Entry.Shadowed {
		parts = append(parts, "shadowed")
	}
	detail := i.Entry.Description
	if detail == "" && i.Entry.Type == domain.TypeAlias {
		detail = i.Entry.Value
	}
	if detail == "" {
		detail = utils.DisplayPath(i.Entry.File)
	}
	parts = append(parts, firstLine(detail))
	return strings.Join(parts, " · ")
}

// FilterValue filters on the entry name.
func (i EntryItem) FilterValue() string { return i.Entry.Name }

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// TypeFilter selects which entry types the list shows.
type TypeFilter int

const (
	FilterAll TypeFilter = iota
	FilterAlias
	FilterFunction
	FilterExecutable
	typeFilterCount
)

// Label is the human-readable name of the filter.
func (f TypeFilter) Label() string {
	switch f {
	case FilterAlias:
		return "aliases"
	case FilterFunction:
		return "functions"
	case FilterExecutable:
		return "executables"
	default:
		return "all"
	}
}

// Next cycles forward (delta 1) or backward (delta -1) through filters.
func (f TypeFilter) Next(delta int) TypeFilter {
	n := int(typeFilterCount)
	return TypeFilter(((int(f)+delta)%n + n) % n)
}

// Matches reports whether an entry passes the filter.
func (f TypeFilter) Matches(e domain.Entry) bool {
	switch f {
	case FilterAlias:
		return e.Type == domain.TypeAlias
	case FilterFunction:
		return e.Type == domain.TypeFunction
	case FilterExecutable:
		return e.Type == domain.TypeExecutable
	default:
		return true
	}
}

// entryItems converts entries passing the filter into list items.
func entryItems(entries []domain.Entry, filter TypeFilter) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		if filter.Matches(e) {
			items = append(items, EntryItem{Entry: e})
		}
	}
	return items
}

// BadgeStyle returns the badge style for an entry type, colored from the theme.
func BadgeStyle(theme Theme, e domain.Entry) lipgloss.Style {
	color := theme.Primary
	switch e.Type {
	case domain.TypeAlias:
		color = theme.Tags
	case domain.TypeFunction:
		color = theme.Flags
	}
	if e.Shadowed {
		color = theme.Muted
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true)
}

// entryDelegate wraps the shared default delegate and prefixes each
// description with a colored type badge.
type entryDelegate struct {
	list.DefaultDelegate
	theme Theme
}

func newEntryDelegate(theme Theme, spacing string) entryDelegate {
	base, _ := NewListDelegate(theme, ListDelegateOptions{Spacing: spacing}).(list.DefaultDelegate)
	return entryDelegate{DefaultDelegate: base, theme: theme}
}

// Render styles the badge and detail separately so the badge color is not
// lost inside the delegate's description style, then defers to the default
// delegate for layout, truncation, and filter-match highlighting.
func (d entryDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	entryItem, ok := item.(EntryItem)
	if !ok || !d.ShowDescription {
		d.DefaultDelegate.Render(w, m, index, item)
		return
	}
	descStyle := d.Styles.NormalDesc
	if index == m.Index() && m.FilterState() != list.Filtering {
		descStyle = d.Styles.SelectedDesc
	}
	rest := lipgloss.NewStyle().Foreground(descStyle.GetForeground())
	badge := BadgeStyle(d.theme, entryItem.Entry).Render(entryItem.Entry.TypeLabel())
	d.DefaultDelegate.Render(w, m, index, styledItem{
		EntryItem:   entryItem,
		description: badge + rest.Render(" "+entryItem.Detail()),
	})
}

// styledItem overrides an item's description with pre-styled text.
type styledItem struct {
	EntryItem
	description string
}

func (s styledItem) Description() string { return s.description }
