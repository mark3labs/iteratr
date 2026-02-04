package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockScrollItem implements ScrollItem for testing scroll position
type mockScrollItem struct {
	id     string
	lines  int
	width  int
	height int
}

func (m *mockScrollItem) ID() string {
	return m.id
}

func (m *mockScrollItem) Render(width int) string {
	m.width = width
	// Create N lines of text
	var b strings.Builder
	for i := 0; i < m.lines; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Item %s line %d", m.id, i+1))
	}
	m.height = m.lines
	return b.String()
}

func (m *mockScrollItem) Height() int {
	return m.height
}

func newMockItem(id string, lines int) *mockScrollItem {
	return &mockScrollItem{id: id, lines: lines}
}

// TestScrollList_InitialPosition tests the initial scroll position
func TestScrollList_InitialPosition(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Empty list starts at top
	assert.Equal(t, 0, sl.offsetIdx, "empty list should have offsetIdx=0")
	assert.Equal(t, 0, sl.offsetLine, "empty list should have offsetLine=0")
	assert.True(t, sl.AtBottom(), "empty list should be at bottom")
	assert.Equal(t, 0.0, sl.ScrollPercent(), "empty list should have 0% scroll")

	// Add items
	items := []ScrollItem{
		newMockItem("item1", 3),
		newMockItem("item2", 3),
		newMockItem("item3", 3),
	}
	sl.SetItems(items)

	// With auto-scroll enabled, should be at bottom (all fits in viewport)
	assert.Equal(t, 0, sl.offsetIdx, "all items fit, should be at top")
	assert.Equal(t, 0, sl.offsetLine, "all items fit, should have no line offset")
	assert.True(t, sl.AtBottom(), "should be at bottom when all items fit")
	assert.Equal(t, 1.0, sl.ScrollPercent(), "all visible should be 100%")
}

// TestScrollList_GotoBottom_Position tests GotoBottom scroll position
func TestScrollList_GotoBottom_Position(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create items totaling 20 lines (exceeds viewport height of 10)
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
		newMockItem("item4", 5),
	}
	sl.SetItems(items)
	sl.GotoTop() // Start at top

	assert.Equal(t, 0, sl.offsetIdx, "should start at top")
	assert.Equal(t, 0, sl.offsetLine, "should have no offset at top")
	assert.False(t, sl.AtBottom(), "should not be at bottom")

	// Go to bottom
	sl.GotoBottom()

	// Should show last 10 lines (items 2-4 visible)
	assert.True(t, sl.AtBottom(), "should be at bottom after GotoBottom")
	assert.Equal(t, 1.0, sl.ScrollPercent(), "should be at 100% when at bottom")

	// Verify current offset is showing last 10 lines
	totalLines := sl.TotalLineCount()
	assert.Equal(t, 20, totalLines, "total lines should be 20")

	currentOffset := sl.currentOffsetInLines()
	assert.Equal(t, 10, currentOffset, "should be at line 10 to show last 10 lines")
}

// TestScrollList_GotoTop_Position tests GotoTop scroll position
func TestScrollList_GotoTop_Position(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)
	sl.GotoBottom() // Start at bottom

	assert.True(t, sl.AtBottom(), "should start at bottom")

	// Go to top
	sl.GotoTop()

	assert.Equal(t, 0, sl.offsetIdx, "should be at first item")
	assert.Equal(t, 0, sl.offsetLine, "should have no line offset")
	assert.Equal(t, 0, sl.currentOffsetInLines(), "current offset should be 0")
	assert.Equal(t, 0.0, sl.ScrollPercent(), "scroll percent should be 0% at top")
	assert.False(t, sl.AtBottom(), "should not be at bottom")
}

// TestScrollList_ScrollBy_PositionTracking tests scroll position after ScrollBy operations
func TestScrollList_ScrollBy_PositionTracking(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// 3 items, 5 lines each = 15 total lines
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Initially at top (offset=0)
	assert.Equal(t, 0, sl.currentOffsetInLines(), "should start at line 0")
	assert.Equal(t, 0.0, sl.ScrollPercent(), "should be at 0%")

	// Scroll down 3 lines
	sl.ScrollBy(3)
	assert.Equal(t, 3, sl.currentOffsetInLines(), "should be at line 3")
	// Percent = 3 / (15-10) = 3/5 = 0.6
	assert.InDelta(t, 0.6, sl.ScrollPercent(), 0.01, "should be at 60%")

	// Scroll down 2 more lines (total 5)
	sl.ScrollBy(2)
	assert.Equal(t, 5, sl.currentOffsetInLines(), "should be at line 5")
	// Percent = 5 / 5 = 1.0
	assert.Equal(t, 1.0, sl.ScrollPercent(), "should be at 100%")
	assert.True(t, sl.AtBottom(), "should be at bottom")

	// Scroll up 2 lines
	sl.ScrollBy(-2)
	assert.Equal(t, 3, sl.currentOffsetInLines(), "should be back at line 3")
	assert.InDelta(t, 0.6, sl.ScrollPercent(), 0.01, "should be at 60%")
	assert.False(t, sl.AtBottom(), "should not be at bottom")
}

// TestScrollList_ScrollBy_Boundaries tests scrolling at boundaries
func TestScrollList_ScrollBy_Boundaries(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create items that exceed viewport (15 total lines)
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)

	// Try scrolling up from top (should stay at top)
	sl.GotoTop()
	sl.ScrollBy(-10)
	assert.Equal(t, 0, sl.currentOffsetInLines(), "should stay at 0 when scrolling up from top")
	assert.Equal(t, 0.0, sl.ScrollPercent(), "should remain at 0%")

	// Try scrolling down past bottom (should stop at bottom)
	sl.GotoTop()
	sl.ScrollBy(100)
	assert.True(t, sl.AtBottom(), "should be at bottom")
	assert.Equal(t, 1.0, sl.ScrollPercent(), "should be at 100%")
}

// TestScrollList_ScrollPercent_Incremental tests scroll percentage during incremental scrolling
func TestScrollList_ScrollPercent_Incremental(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// 10 items, 2 lines each = 20 total lines
	// Viewport shows 10 lines, so max offset is 10
	items := make([]ScrollItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = newMockItem(fmt.Sprintf("item%d", i), 2)
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Test scroll percentages at different positions
	testCases := []struct {
		scrollLines      int
		expectedPct      float64
		expectedAtBottom bool
	}{
		{0, 0.0, false}, // At top: 0/10 = 0%
		{2, 0.2, false}, // 2 lines down: 2/10 = 20%
		{5, 0.5, false}, // 5 lines down: 5/10 = 50%
		{8, 0.8, false}, // 8 lines down: 8/10 = 80%
		{10, 1.0, true}, // 10 lines down: 10/10 = 100% (at bottom)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("scroll_%d_lines", tc.scrollLines), func(t *testing.T) {
			sl.GotoTop()
			sl.ScrollBy(tc.scrollLines)

			assert.Equal(t, tc.scrollLines, sl.currentOffsetInLines(), "offset should match scroll amount")
			assert.InDelta(t, tc.expectedPct, sl.ScrollPercent(), 0.01, "scroll percent mismatch")
			assert.Equal(t, tc.expectedAtBottom, sl.AtBottom(), "AtBottom mismatch")
		})
	}
}

// TestScrollList_ScrollToItem_Position tests scroll position after ScrollToItem
func TestScrollList_ScrollToItem_Position(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// 5 items, 3 lines each = 15 total lines
	items := []ScrollItem{
		newMockItem("item0", 3),
		newMockItem("item1", 3),
		newMockItem("item2", 3),
		newMockItem("item3", 3),
		newMockItem("item4", 3),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Scroll to item 2 (starts at line 6)
	sl.ScrollToItem(2)

	// Item 2 starts at line 6, but since viewport is 10 lines and item is 3 lines,
	// item should be visible without scrolling (0-9 visible, item at 6-8)
	assert.Equal(t, 0, sl.currentOffsetInLines(), "item 2 already visible, no scroll needed")

	// Scroll to item 4 (starts at line 12)
	sl.ScrollToItem(4)

	// Item 4 is at lines 12-14, viewport shows 10 lines
	// Should scroll so item 4 is visible: offset should be at least 5 (showing 5-14)
	currentOffset := sl.currentOffsetInLines()
	assert.True(t, currentOffset >= 5, "should scroll to show item 4")
	assert.True(t, sl.AtBottom(), "scrolling to last item should reach bottom")
}

// TestScrollList_ScrollToItem_AlreadyVisible tests that ScrollToItem doesn't scroll if item visible
func TestScrollList_ScrollToItem_AlreadyVisible(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("item0", 2),
		newMockItem("item1", 2),
		newMockItem("item2", 2),
		newMockItem("item3", 2),
		newMockItem("item4", 2),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// All items fit in viewport (10 lines total, viewport is 10)
	// ScrollToItem should not change position for any item
	for i := 0; i < 5; i++ {
		initialOffset := sl.currentOffsetInLines()
		sl.ScrollToItem(i)
		assert.Equal(t, initialOffset, sl.currentOffsetInLines(),
			"ScrollToItem should not change position when item already visible")
	}
}

// TestScrollList_KeyboardScroll_PositionUpdates tests that keyboard scrolling updates position
func TestScrollList_KeyboardScroll_PositionUpdates(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetFocused(true) // Must be focused to handle keyboard

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
		newMockItem("item4", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Press down arrow (scroll down 1 line)
	sl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 1, sl.currentOffsetInLines(), "down arrow should scroll 1 line")

	// Press 'j' (vim-style down)
	sl.Update(tea.KeyPressMsg{Code: 'j'})
	assert.Equal(t, 2, sl.currentOffsetInLines(), "j key should scroll 1 line")

	// Press up arrow (scroll up 1 line)
	sl.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Equal(t, 1, sl.currentOffsetInLines(), "up arrow should scroll up 1 line")

	// Press 'k' (vim-style up)
	sl.Update(tea.KeyPressMsg{Code: 'k'})
	assert.Equal(t, 0, sl.currentOffsetInLines(), "k key should scroll up 1 line")

	// Press pgdown (scroll down viewport height)
	sl.Update(tea.KeyPressMsg{Text: "pgdown"})
	assert.Equal(t, 10, sl.currentOffsetInLines(), "pgdown should scroll viewport height")

	// Press pgup (scroll up viewport height)
	sl.Update(tea.KeyPressMsg{Text: "pgup"})
	assert.Equal(t, 0, sl.currentOffsetInLines(), "pgup should scroll up viewport height")

	// Press end (go to bottom)
	sl.Update(tea.KeyPressMsg{Text: "end"})
	assert.True(t, sl.AtBottom(), "end key should go to bottom")

	// Press home (go to top)
	sl.Update(tea.KeyPressMsg{Text: "home"})
	assert.Equal(t, 0, sl.currentOffsetInLines(), "home key should go to top")
}

// TestScrollList_MultiLineItem_PartialScroll tests scrolling with multi-line items
func TestScrollList_MultiLineItem_PartialScroll(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// First item is 15 lines (exceeds viewport)
	items := []ScrollItem{
		newMockItem("bigitem", 15),
		newMockItem("item2", 3),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Should be at start of big item
	assert.Equal(t, 0, sl.offsetIdx, "should be at first item")
	assert.Equal(t, 0, sl.offsetLine, "should be at line 0 of first item")

	// Scroll down 5 lines (still within first item)
	sl.ScrollBy(5)
	assert.Equal(t, 0, sl.offsetIdx, "should still be at first item")
	assert.Equal(t, 5, sl.offsetLine, "should be at line 5 of first item")
	assert.Equal(t, 5, sl.currentOffsetInLines(), "current offset should be 5")

	// Scroll down 10 more lines (should move to second item)
	sl.ScrollBy(10)
	assert.Equal(t, 1, sl.offsetIdx, "should have moved to second item")
	assert.Equal(t, 15, sl.currentOffsetInLines(), "should be at line 15 overall")
}

// TestScrollList_AutoScroll_DisablesOnScrollUp tests that scrolling up disables auto-scroll
func TestScrollList_AutoScroll_DisablesOnScrollUp(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetFocused(true)

	items := []ScrollItem{
		newMockItem("item1", 10),
		newMockItem("item2", 10),
	}
	sl.SetItems(items)

	// Auto-scroll should be enabled initially
	assert.True(t, sl.autoScroll, "auto-scroll should be enabled initially")

	// Scroll up should disable auto-scroll
	sl.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.False(t, sl.autoScroll, "auto-scroll should be disabled after scrolling up")

	// Scroll to bottom should re-enable auto-scroll
	sl.Update(tea.KeyPressMsg{Text: "end"})
	assert.True(t, sl.autoScroll, "auto-scroll should be re-enabled at bottom")
}

// TestScrollList_EmptyList_SafeScrolling tests that scrolling operations are safe on empty list
func TestScrollList_EmptyList_SafeScrolling(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// All scroll operations should be safe on empty list
	require.NotPanics(t, func() {
		sl.GotoTop()
		sl.GotoBottom()
		sl.ScrollBy(10)
		sl.ScrollBy(-10)
		sl.ScrollToItem(0)
		sl.ScrollPercent()
		sl.AtBottom()
		sl.TotalLineCount()
		sl.currentOffsetInLines()
	}, "scroll operations should not panic on empty list")

	// Position should remain at 0
	assert.Equal(t, 0, sl.offsetIdx, "empty list should stay at offsetIdx=0")
	assert.Equal(t, 0, sl.offsetLine, "empty list should stay at offsetLine=0")
}

// TestScrollList_SingleItem_FitsInViewport tests scrolling when single item fits
func TestScrollList_SingleItem_FitsInViewport(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("single", 5),
	}
	sl.SetItems(items)

	// Everything fits, should be at top/bottom simultaneously
	assert.Equal(t, 0, sl.currentOffsetInLines(), "should be at top")
	assert.True(t, sl.AtBottom(), "should also be at bottom (all visible)")
	assert.Equal(t, 1.0, sl.ScrollPercent(), "should be 100% (all visible)")

	// Scrolling should have no effect
	sl.ScrollBy(5)
	assert.Equal(t, 0, sl.currentOffsetInLines(), "scroll should have no effect")

	sl.ScrollBy(-5)
	assert.Equal(t, 0, sl.currentOffsetInLines(), "scroll should have no effect")
}

// TestScrollList_OffsetClamping tests that offset is properly clamped
func TestScrollList_OffsetClamping(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
	}
	sl.SetItems(items)

	// Manually set invalid offsets and trigger clamp
	sl.offsetIdx = 10   // Invalid: beyond items
	sl.offsetLine = 100 // Invalid: beyond item height
	sl.clampOffset()

	// Should be clamped to valid range
	assert.Equal(t, 1, sl.offsetIdx, "offsetIdx should be clamped to last item")
	assert.True(t, sl.offsetLine >= 0 && sl.offsetLine < 5, "offsetLine should be clamped to item height")
}
