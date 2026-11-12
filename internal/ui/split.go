package ui

import "github.com/charmbracelet/lipgloss"

// SplitLayout renders two bordered panes side by side with
// lipgloss.JoinHorizontal. Sizes are outer dimensions, borders included.
type SplitLayout struct {
	Width  int
	Height int
	// LeftRatio is the share of the width given to the left pane (0–1).
	LeftRatio float64
	// LeftMin and LeftMax clamp the left pane's outer width.
	LeftMin, LeftMax int
	Theme            Theme
}

// NewSplitLayout returns a layout with list-friendly defaults.
func NewSplitLayout(theme Theme) SplitLayout {
	return SplitLayout{LeftRatio: 0.4, LeftMin: 30, LeftMax: 60, Theme: theme}
}

// ShouldSplit reports whether a breakpoint is wide enough for two panes.
func ShouldSplit(bp Breakpoint) bool {
	return bp >= BreakpointLG
}

func (s SplitLayout) paneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Theme.Border).
		Padding(0, 1)
}

// widths returns the outer width of each pane.
func (s SplitLayout) widths() (left, right int) {
	left = int(float64(s.Width) * s.LeftRatio)
	if s.LeftMin > 0 {
		left = max(left, s.LeftMin)
	}
	if s.LeftMax > 0 {
		left = min(left, s.LeftMax)
	}
	left = min(left, s.Width/2)
	return left, s.Width - left
}

// ContentSizes returns the inner (content) width of each pane and the
// shared inner height, after borders and padding.
func (s SplitLayout) ContentSizes() (leftWidth, rightWidth, height int) {
	style := s.paneStyle()
	left, right := s.widths()
	h := style.GetHorizontalFrameSize()
	return max(left-h, 1), max(right-h, 1), max(s.Height-style.GetVerticalFrameSize(), 1)
}

// Render joins the two pane contents horizontally inside bordered frames.
func (s SplitLayout) Render(left, right string) string {
	lw, rw, h := s.ContentSizes()
	style := s.paneStyle()
	return lipgloss.JoinHorizontal(lipgloss.Top,
		style.Width(lw+style.GetHorizontalPadding()).Height(h).MaxHeight(h+style.GetVerticalBorderSize()).Render(left),
		style.Width(rw+style.GetHorizontalPadding()).Height(h).MaxHeight(h+style.GetVerticalBorderSize()).Render(right),
	)
}

// RenderSingle renders one full-width pane with the same frame.
func (s SplitLayout) RenderSingle(content string) string {
	style := s.paneStyle()
	w := max(s.Width-style.GetHorizontalFrameSize(), 1)
	h := max(s.Height-style.GetVerticalFrameSize(), 1)
	return style.Width(w + style.GetHorizontalPadding()).Height(h).MaxHeight(h + style.GetVerticalBorderSize()).Render(content)
}

// SingleContentSize returns the inner size of a full-width pane.
func (s SplitLayout) SingleContentSize() (width, height int) {
	style := s.paneStyle()
	return max(s.Width-style.GetHorizontalFrameSize(), 1), max(s.Height-style.GetVerticalFrameSize(), 1)
}
