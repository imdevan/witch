package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"unicode"

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
	// query is the search term, highlighted in addition to the entry name.
	query    string
	files    map[string][]string
	readFile func(string) ([]string, error)
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

// SetQuery sets the search term to highlight alongside the entry name.
// The scroll position is kept so typing doesn't jump the view.
func (p *Preview) SetQuery(q string) {
	q = strings.TrimSpace(q)
	if q == p.query {
		return
	}
	p.query = q
	if p.entry == nil {
		return
	}
	if lines, err := p.lines(*p.entry); err == nil {
		offset := p.viewport.YOffset
		p.viewport.SetContent(p.renderLines(lines))
		p.viewport.SetYOffset(offset)
	}
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
	match := lipgloss.NewStyle().Background(p.theme.Highlight).Foreground(lipgloss.Color("0"))
	name := ""
	if p.entry != nil {
		name = p.entry.Name
	}

	var b strings.Builder
	for i, line := range lines {
		n := i + 1
		line = strings.ReplaceAll(line, "\t", strings.Repeat(" ", tabWidth))
		number := fmt.Sprintf("%*d", digits, n)
		if n >= p.HighlightStart && n <= p.HighlightEnd && p.HighlightStart > 0 {
			b.WriteString(marker.Render(number+" ▌") + styleMatches(line, name, p.query, highlight, match))
		} else {
			b.WriteString(gutter.Render(number+"  ") + styleMatches(line, name, p.query, text, match))
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// styleMatches renders line in base, with occurrences of name (as a whole
// word) and query (case-insensitive substring) styled with match on top.
func styleMatches(line, name, query string, base, match lipgloss.Style) string {
	indices := MatchIndices(line, name, query)
	if len(indices) == 0 {
		return base.Render(line)
	}
	return lipgloss.StyleRunes(line, indices, match.Inherit(base).Inline(true), base.Inline(true))
}

// MatchIndices returns the rune indices in line covered by whole-word
// occurrences of name and case-insensitive occurrences of query. The query
// is skipped when it is empty or the same as the name.
func MatchIndices(line, name, query string) []int {
	runes := []rune(line)
	marked := make([]bool, len(runes))
	if name != "" {
		markAll(runes, []rune(name), marked, true)
	}
	if query != "" && !strings.EqualFold(query, name) {
		lower := []rune(strings.ToLower(line))
		if len(lower) == len(runes) {
			markAll(lower, []rune(strings.ToLower(query)), marked, false)
		}
	}
	var indices []int
	for i, m := range marked {
		if m {
			indices = append(indices, i)
		}
	}
	return indices
}

func markAll(text, word []rune, marked []bool, wholeWord bool) {
	if len(word) == 0 {
		return
	}
	for i := 0; i+len(word) <= len(text); i++ {
		if !runesEqual(text[i:i+len(word)], word) {
			continue
		}
		end := i + len(word)
		if wholeWord && ((i > 0 && isWordRune(text[i-1])) || (end < len(text) && isWordRune(text[end]))) {
			continue
		}
		for j := i; j < end; j++ {
			marked[j] = true
		}
	}
}

func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isWordRune reports whether r can be part of a command name.
func isWordRune(r rune) bool {
	return r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
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
