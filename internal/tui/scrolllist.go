package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ScrollItem represents a single item in a ScrollList.
// This is an alias for MessageItem since they share the same interface.
type ScrollItem interface {
	// ID returns the unique identifier for this item.
	ID() string
	// Render returns the rendered string representation at the given width.
	Render(width int) string
	// Height returns the number of lines this item occupies (0 if not yet rendered).
	Height() int
}

// ScrollList is a lazy-rendering scrollable list that only renders visible items.
// Unlike viewport which re-processes entire content on every SetContent(),
// ScrollList maintains an offset-based view into a list of items and only
// renders what's visible in the current viewport.
type ScrollList struct {
	items       []ScrollItem // All items in the list
	offsetIdx   int          // Index of the first visible item
	offsetLine  int          // Number of lines to skip from the first visible item
	width       int          // Viewport width
	height      int          // Viewport height
	itemGap     int          // Number of blank lines between items (0 = no gap)
	autoScroll  bool         // Whether to auto-scroll to bottom on content changes
	focused     bool         // Whether this list has keyboard focus
	selectedIdx int          // Index of selected item (-1 = no selection)
}

// NewScrollList creates a new ScrollList with the given width and height.
func NewScrollList(width, height int) *ScrollList {
	return &ScrollList{
		items:       make([]ScrollItem, 0),
		offsetIdx:   0,
		offsetLine:  0,
		width:       width,
		height:      height,
		autoScroll:  true,
		focused:     false,
		selectedIdx: -1,
	}
}

// SetItems replaces all items in the list.
func (s *ScrollList) SetItems(items []ScrollItem) {
	s.items = items
	// Clamp offset to new bounds
	s.clampOffset()
}

// AppendItem adds a single item to the end of the list.
func (s *ScrollList) AppendItem(item ScrollItem) {
	s.items = append(s.items, item)
	if s.autoScroll {
		s.GotoBottom()
	}
}

// SetWidth updates the viewport width.
func (s *ScrollList) SetWidth(width int) {
	s.width = width
}

// SetHeight updates the viewport height.
func (s *ScrollList) SetHeight(height int) {
	s.height = height
	s.clampOffset()
}

// SetItemGap sets the number of blank lines rendered between items.
func (s *ScrollList) SetItemGap(gap int) {
	s.itemGap = gap
}

// SetAutoScroll enables or disables auto-scrolling to bottom.
func (s *ScrollList) SetAutoScroll(enabled bool) {
	s.autoScroll = enabled
}

// SetFocused sets the focus state of the list.
func (s *ScrollList) SetFocused(focused bool) {
	s.focused = focused
}

// SetSelected sets the selected item index.
// Pass -1 to clear selection.
func (s *ScrollList) SetSelected(idx int) {
	if idx < -1 || idx >= len(s.items) {
		s.selectedIdx = -1
	} else {
		s.selectedIdx = idx
	}
}

// SelectedIdx returns the current selected item index (-1 if no selection).
func (s *ScrollList) SelectedIdx() int {
	return s.selectedIdx
}

// View returns the rendered view of visible items.
// Only renders items that are visible within the viewport height.
func (s *ScrollList) View() string {
	if len(s.items) == 0 {
		return ""
	}

	var result strings.Builder
	linesRendered := 0

	// Start from offsetIdx and render items until we fill the viewport height
	for i := s.offsetIdx; i < len(s.items) && linesRendered < s.height; i++ {
		item := s.items[i]

		// Render the item
		rendered := item.Render(s.width)

		// For the first visible item, skip offsetLine lines
		if i == s.offsetIdx && s.offsetLine > 0 {
			lines := strings.Split(rendered, "\n")
			if s.offsetLine < len(lines) {
				// Skip first offsetLine lines
				rendered = strings.Join(lines[s.offsetLine:], "\n")
			} else {
				// Skip entire item if offset is beyond its height
				continue
			}
		}

		// Apply selection styling if this item is selected
		if i == s.selectedIdx {
			// Add selection indicator prefix
			rendered = theme.Current().S().TaskSelected.Render("▸ ") + rendered
		}

		// Count lines in rendered output
		itemLines := strings.Count(rendered, "\n") + 1
		if linesRendered+itemLines > s.height {
			// Truncate to fit remaining space
			lines := strings.Split(rendered, "\n")
			remainingLines := s.height - linesRendered
			if remainingLines > 0 {
				rendered = strings.Join(lines[:remainingLines], "\n")
			} else {
				break
			}
		}

		result.WriteString(rendered)
		linesRendered += strings.Count(rendered, "\n") + 1

		// Add newline + gap between items (if not the last visible item)
		if linesRendered < s.height && i < len(s.items)-1 {
			result.WriteString("\n")
			// Add gap blank lines between items
			for g := 0; g < s.itemGap && linesRendered+1 < s.height; g++ {
				result.WriteString("\n")
				linesRendered++
			}
		}
	}

	return result.String()
}

// ScrollBy scrolls the viewport by the given number of lines.
// Positive values scroll down, negative values scroll up.
func (s *ScrollList) ScrollBy(lines int) {
	if lines == 0 {
		return
	}

	if lines > 0 {
		// Scroll down
		for lines > 0 && s.offsetIdx < len(s.items) {
			currentItem := s.items[s.offsetIdx]
			itemHeight := currentItem.Height()
			if itemHeight == 0 {
				// Not rendered yet, render to get height
				currentItem.Render(s.width)
				itemHeight = currentItem.Height()
			}

			remainingLines := itemHeight - s.offsetLine
			if lines >= remainingLines {
				// Move past current item
				s.offsetIdx++
				s.offsetLine = 0
				lines -= remainingLines
				// Consume gap lines between items
				if s.itemGap > 0 && s.offsetIdx < len(s.items) {
					if lines >= s.itemGap {
						lines -= s.itemGap
					} else {
						lines = 0
					}
				}
			} else {
				// Scroll within current item
				s.offsetLine += lines
				lines = 0
			}
		}
	} else {
		// Scroll up
		lines = -lines
		for lines > 0 && (s.offsetIdx > 0 || s.offsetLine > 0) {
			if s.offsetLine >= lines {
				// Scroll within current item
				s.offsetLine -= lines
				lines = 0
			} else {
				// Move to previous item
				lines -= s.offsetLine
				s.offsetLine = 0
				if s.offsetIdx > 0 {
					// Consume gap lines between items
					if s.itemGap > 0 {
						if lines > s.itemGap {
							lines -= s.itemGap
						} else {
							lines = 0
							continue
						}
					}
					s.offsetIdx--
					prevItem := s.items[s.offsetIdx]
					prevHeight := prevItem.Height()
					if prevHeight == 0 {
						prevItem.Render(s.width)
						prevHeight = prevItem.Height()
					}
					s.offsetLine = prevHeight - 1
					if s.offsetLine < 0 {
						s.offsetLine = 0
					}
					lines--
				}
			}
		}
	}

	s.clampOffset()
}

// GotoBottom scrolls to the bottom of the list.
func (s *ScrollList) GotoBottom() {
	if len(s.items) == 0 {
		return
	}

	// Set offset to show the last item at the bottom of viewport
	totalLines := s.TotalLineCount()
	if totalLines <= s.height {
		// Everything fits, go to top
		s.offsetIdx = 0
		s.offsetLine = 0
		return
	}

	// Calculate offset to show last s.height lines
	targetLine := totalLines - s.height
	currentLine := 0

	for i := 0; i < len(s.items); i++ {
		item := s.items[i]
		itemHeight := item.Height()
		if itemHeight == 0 {
			item.Render(s.width)
			itemHeight = item.Height()
		}

		if currentLine+itemHeight > targetLine {
			// This item contains the target line
			s.offsetIdx = i
			s.offsetLine = targetLine - currentLine
			return
		}

		currentLine += itemHeight
		// Add gap after item (except the last)
		if s.itemGap > 0 && i < len(s.items)-1 {
			currentLine += s.itemGap
		}
	}

	// Fallback: show last item
	s.offsetIdx = len(s.items) - 1
	s.offsetLine = 0
}

// GotoTop scrolls to the top of the list.
func (s *ScrollList) GotoTop() {
	s.offsetIdx = 0
	s.offsetLine = 0
}

// ScrollToItem scrolls the minimum amount needed to make the item at idx visible.
// If the item is already fully visible, no scrolling occurs.
func (s *ScrollList) ScrollToItem(idx int) {
	if idx < 0 || idx >= len(s.items) {
		return
	}

	// Calculate the line offset of the target item
	itemStartLine := 0
	for i := 0; i < idx; i++ {
		h := s.items[i].Height()
		if h == 0 {
			s.items[i].Render(s.width)
			h = s.items[i].Height()
		}
		itemStartLine += h
		if s.itemGap > 0 {
			itemStartLine += s.itemGap
		}
	}

	itemHeight := s.items[idx].Height()
	if itemHeight == 0 {
		s.items[idx].Render(s.width)
		itemHeight = s.items[idx].Height()
	}
	itemEndLine := itemStartLine + itemHeight

	currentOffset := s.currentOffsetInLines()

	// If item starts above viewport, scroll up to show it
	if itemStartLine < currentOffset {
		// Set offset to show this item at the top
		s.offsetIdx = idx
		s.offsetLine = 0
		return
	}

	// If item ends below viewport, scroll down to show it
	if itemEndLine > currentOffset+s.height {
		// Scroll so the item's bottom aligns with viewport bottom
		targetOffset := itemEndLine - s.height
		if targetOffset < 0 {
			targetOffset = 0
		}
		// Find which item contains the target offset line
		lineCount := 0
		for i := 0; i < len(s.items); i++ {
			h := s.items[i].Height()
			if h == 0 {
				s.items[i].Render(s.width)
				h = s.items[i].Height()
			}
			if lineCount+h > targetOffset {
				s.offsetIdx = i
				s.offsetLine = targetOffset - lineCount
				return
			}
			lineCount += h
			if s.itemGap > 0 && i < len(s.items)-1 {
				lineCount += s.itemGap
			}
		}
	}
	// Item is already visible, no scrolling needed
}

// AtBottom returns true if the viewport is scrolled to the bottom.
func (s *ScrollList) AtBottom() bool {
	if len(s.items) == 0 {
		return true
	}

	totalLines := s.TotalLineCount()
	currentOffset := s.currentOffsetInLines()

	// At bottom if we're showing the last s.height lines
	return currentOffset+s.height >= totalLines
}

// TotalLineCount returns the total number of lines across all items,
// including gap lines between items.
func (s *ScrollList) TotalLineCount() int {
	total := 0
	for _, item := range s.items {
		h := item.Height()
		if h == 0 {
			// Not rendered yet, render to get height
			item.Render(s.width)
			h = item.Height()
		}
		total += h
	}
	// Add gap lines between items (N-1 gaps for N items)
	if len(s.items) > 1 && s.itemGap > 0 {
		total += (len(s.items) - 1) * s.itemGap
	}
	return total
}

// ScrollPercent returns the current scroll position as a percentage (0.0 to 1.0).
func (s *ScrollList) ScrollPercent() float64 {
	if len(s.items) == 0 {
		return 0.0
	}

	totalLines := s.TotalLineCount()
	if totalLines <= s.height {
		return 1.0 // Everything visible
	}

	currentOffset := s.currentOffsetInLines()
	maxOffset := totalLines - s.height

	if maxOffset <= 0 {
		return 1.0
	}

	pct := float64(currentOffset) / float64(maxOffset)
	if pct > 1.0 {
		pct = 1.0
	}
	if pct < 0.0 {
		pct = 0.0
	}

	return pct
}

// Update handles messages for the scroll list.
// Only processes keyboard events when focused is true.
func (s *ScrollList) Update(msg tea.Msg) tea.Cmd {
	if !s.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			// Scroll up by one line (arrows primary, j/k backup)
			s.ScrollBy(-1)
			s.autoScroll = false
		case "down", "j":
			// Scroll down by one line (arrows primary, j/k backup)
			s.ScrollBy(1)
			if s.AtBottom() {
				s.autoScroll = true
			}
		case "pgup":
			s.ScrollBy(-s.height)
			s.autoScroll = false
		case "pgdown":
			s.ScrollBy(s.height)
			if s.AtBottom() {
				s.autoScroll = true
			}
		case "home":
			s.GotoTop()
			s.autoScroll = false
		case "end":
			s.GotoBottom()
			s.autoScroll = true
		}
	}

	return nil
}

// currentOffsetInLines returns the current scroll offset in lines.
func (s *ScrollList) currentOffsetInLines() int {
	offset := 0
	for i := 0; i < s.offsetIdx && i < len(s.items); i++ {
		h := s.items[i].Height()
		if h == 0 {
			s.items[i].Render(s.width)
			h = s.items[i].Height()
		}
		offset += h
		// Add gap after each item (except the last one before offsetIdx)
		if s.itemGap > 0 {
			offset += s.itemGap
		}
	}
	offset += s.offsetLine
	return offset
}

// clampOffset ensures offset is within valid bounds.
func (s *ScrollList) clampOffset() {
	if len(s.items) == 0 {
		s.offsetIdx = 0
		s.offsetLine = 0
		return
	}

	// Clamp offsetIdx
	if s.offsetIdx >= len(s.items) {
		s.offsetIdx = len(s.items) - 1
	}
	if s.offsetIdx < 0 {
		s.offsetIdx = 0
	}

	// Clamp offsetLine within current item
	if s.offsetIdx < len(s.items) {
		item := s.items[s.offsetIdx]
		h := item.Height()
		if h == 0 {
			item.Render(s.width)
			h = item.Height()
		}
		if s.offsetLine >= h {
			s.offsetLine = h - 1
			if s.offsetLine < 0 {
				s.offsetLine = 0
			}
		}
		if s.offsetLine < 0 {
			s.offsetLine = 0
		}
	}
}
