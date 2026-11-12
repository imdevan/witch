package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/sources"
	"github.com/imdevan/witch/internal/utils"
)

// previewHeaderHeight is the number of lines above the viewport.
const previewHeaderHeight = 3

// tabWidth is how many spaces a tab expands to in the preview.
const tabWidth = 4

// Preview shows the file that defines an entry in a bubbles viewport, with
// the definition highlighted and scrolled into view.
type Preview struct {
	viewport viewport.Model
	theme    Theme
	entry    *domain.Entry
	width    int
	height   int
	// HighlightStart and HighlightEnd are the 1-indexed highlighted lines
	// (0 when nothing is highlighted).
	HighlightStart, HighlightEnd int
	files                        map[string][]string
	readFile                     func(string) ([]string, error)
}

// NewPreview creates an empty preview pane.
func NewPreview(theme Theme) Preview {
	vp := viewport.New(0, 0)
	vp.SetHorizontalStep(tabWidth)
	return Preview{
		viewport: vp,
		theme:    theme,
		files:    map[string][]string{},
		readFile: readTextLines,
	}
}

// SetSize sets the outer size of the pane, header included.
func (p *Preview) SetSize(width, height int) {
	p.width, p.height = width, height
	p.viewport.Width = width
	p.viewport.Height = max(height-previewHeaderHeight, 1)
	p.render()
}

// SetEntry shows a new entry, reusing cached file contents.
func (p *Preview) SetEntry(e *domain.Entry) {
	if e != nil && p.entry != nil && *e == *p.entry {
		return
	}
	p.entry = e
	p.render()
}

// Entry returns the entry being previewed.
func (p Preview) Entry() *domain.Entry { return p.entry }

// Update forwards scrolling keys to the viewport.
func (p Preview) Update(msg tea.Msg) (Preview, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// ScrollDown scrolls the preview down n lines (negative scrolls up).
func (p *Preview) ScrollDown(n int) {
	if n < 0 {
		p.viewport.ScrollUp(-n)
		return
	}
	p.viewport.ScrollDown(n)
}

// HalfPage returns half the viewport height, for half-page scrolling.
func (p Preview) HalfPage() int { return max(p.viewport.Height/2, 1) }

// YOffset is the first visible content line (0-indexed).
func (p Preview) YOffset() int { return p.viewport.YOffset }

// View renders the header and the viewport.
func (p Preview) View() string {
	return p.header() + "\n" + p.viewport.View()
}

func (p Preview) header() string {
	if p.entry == nil {
		return lipgloss.NewStyle().Foreground(p.theme.Muted).Render("Nothing selected") + "\n\n"
	}
	e := p.entry
	name := lipgloss.NewStyle().Foreground(p.theme.Headings).Bold(true).Render(e.Name)
	title := name + "  " + BadgeStyle(p.theme, *e).Render(e.TypeLabel())
	if e.Shadowed {
		title += lipgloss.NewStyle().Foreground(p.theme.Muted).Render("  shadowed")
	}
	location := utils.DisplayPath(e.Location())
	if e.Target != "" {
		location += " → " + utils.DisplayPath(e.Target)
	}
	muted := lipgloss.NewStyle().Foreground(p.theme.Muted).MaxWidth(max(p.width, 1))
	return lipgloss.NewStyle().MaxWidth(max(p.width, 1)).Render(title) + "\n" +
		muted.Render(location) + "\n" +
		muted.Render(strings.Repeat("─", max(p.width, 1)))
}

// render rebuilds the viewport content for the current entry and scrolls
// the highlighted definition into view.
func (p *Preview) render() {
	p.HighlightStart, p.HighlightEnd = 0, 0
	if p.entry == nil {
		p.viewport.SetContent("")
		return
	}
	e := *p.entry
	lines, err := p.lines(e)
	if err != nil {
		p.viewport.SetContent(p.details(e, err))
		p.viewport.GotoTop()
		return
	}
	if e.Line > 0 && e.Line <= len(lines) {
		span := 1
		if e.Definition != "" {
			span = strings.Count(e.Definition, "\n") + 1
		}
		p.HighlightStart = e.Line
		p.HighlightEnd = min(e.Line+span-1, len(lines))
	}
	p.viewport.SetContent(p.renderLines(lines))
	p.viewport.SetXOffset(0)
	if p.HighlightStart > 0 {
		// Keep a little context above the definition.
		p.viewport.SetYOffset(max(p.HighlightStart-1-p.viewport.Height/4, 0))
	} else {
		p.viewport.GotoTop()
	}
}

func (p *Preview) lines(e domain.Entry) ([]string, error) {
	if lines, ok := p.files[e.File]; ok {
		return lines, nil
	}
	lines, err := p.readFile(e.File)
	if err != nil {
		return nil, err
	}
	p.files[e.File] = lines
	return lines, nil
}

func (p Preview) renderLines(lines []string) string {
	digits := len(fmt.Sprint(len(lines)))
	gutter := lipgloss.NewStyle().Foreground(p.theme.Muted)
	text := lipgloss.NewStyle().Foreground(p.theme.Text)
	marker := lipgloss.NewStyle().Foreground(p.theme.Primary).Bold(true)
	highlight := lipgloss.NewStyle().Foreground(p.theme.TextHighlight).Bold(true)

	var b strings.Builder
	for i, line := range lines {
		n := i + 1
		line = strings.ReplaceAll(line, "\t", strings.Repeat(" ", tabWidth))
		number := fmt.Sprintf("%*d", digits, n)
		if n >= p.HighlightStart && n <= p.HighlightEnd && p.HighlightStart > 0 {
			b.WriteString(marker.Render(number+" ▌") + highlight.Render(line))
		} else {
			b.WriteString(gutter.Render(number+"  ") + text.Render(line))
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// details describes entries whose file can't be shown, such as binaries.
func (p Preview) details(e domain.Entry, err error) string {
	label := lipgloss.NewStyle().Foreground(p.theme.Muted)
	rows := [][2]string{{"path", e.File}}
	if e.Target != "" {
		rows = append(rows, [2]string{"target", e.Target})
	}
	if info, statErr := os.Stat(e.File); statErr == nil {
		rows = append(rows,
			[2]string{"mode", info.Mode().String()},
			[2]string{"size", humanSize(info.Size())},
		)
	}
	reason := err.Error()
	if errors.Is(err, fs.ErrNotExist) {
		reason = "file not found"
	}
	rows = append(rows, [2]string{"preview", reason})
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(label.Render(fmt.Sprintf("%-8s", r[0])) + r[1] + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// errBinary is shown for files that can't be previewed as text.
var errBinary = errors.New("binary file, not shown")

func readTextLines(path string) ([]string, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	if !sources.IsTextFile(path) {
		return nil, errBinary
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n"), nil
}
