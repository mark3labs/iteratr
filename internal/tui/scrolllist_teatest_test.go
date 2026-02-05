package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestScrollList_Viewport_ExceedsHeight tests rendering when content exceeds viewport height
func TestScrollList_Viewport_ExceedsHeight(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create items totaling 30 lines (exceeds viewport height of 10)
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
		newMockItem("item4", 5),
		newMockItem("item5", 5),
		newMockItem("item6", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Render at top - should show first 10 lines (items 1-2)
	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	if len(lines) > 10 {
		t.Errorf("Rendered %d lines, should not exceed viewport height of 10", len(lines))
	}

	// Verify first item is visible
	if !strings.Contains(rendered, "Item item1 line 1") {
		t.Error("First item should be visible at top")
	}

	// Verify we're not at bottom
	if sl.AtBottom() {
		t.Error("Should not be at bottom when content exceeds viewport")
	}
}

// TestScrollList_Viewport_PartialItemAtBottom tests rendering with partial item visibility at bottom
func TestScrollList_Viewport_PartialItemAtBottom(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Item 1: 3 lines, Item 2: 3 lines, Item 3: 8 lines
	// Total: 14 lines, viewport: 10 lines
	// At top: shows item 1 (3) + item 2 (3) + partial item 3 (4 lines)
	items := []ScrollItem{
		newMockItem("item1", 3),
		newMockItem("item2", 3),
		newMockItem("item3", 8),
	}
	sl.SetItems(items)
	sl.GotoTop()

	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	// Should show exactly 10 lines
	if len(lines) > 10 {
		t.Errorf("Rendered %d lines, should not exceed viewport height of 10", len(lines))
	}

	// Verify item 1 is fully visible
	if !strings.Contains(rendered, "Item item1 line 1") {
		t.Error("Item 1 should be visible")
	}

	// Verify item 2 is fully visible
	if !strings.Contains(rendered, "Item item2 line 1") {
		t.Error("Item 2 should be visible")
	}

	// Verify item 3 is partially visible (first 4 lines only)
	if !strings.Contains(rendered, "Item item3 line 1") {
		t.Error("Item 3 should be partially visible")
	}
	if !strings.Contains(rendered, "Item item3 line 4") {
		t.Error("Item 3 line 4 should be visible")
	}
	if strings.Contains(rendered, "Item item3 line 8") {
		t.Error("Item 3 line 8 should not be visible (below viewport)")
	}
}

// TestScrollList_Viewport_PartialItemAtTop tests rendering with offset within first item
func TestScrollList_Viewport_PartialItemAtTop(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create large first item (15 lines)
	items := []ScrollItem{
		newMockItem("bigitem", 15),
		newMockItem("item2", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Scroll down 5 lines (within first item)
	sl.ScrollBy(5)

	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	// After scrolling down 5 lines, we should see lines 6-15 of bigitem
	if !strings.Contains(rendered, "Item bigitem line 6") {
		t.Errorf("Line 6 should be visible (first visible line), rendered lines: %v", lines)
	}
	if !strings.Contains(rendered, "Item bigitem line 15") {
		t.Errorf("Line 15 should be visible (within viewport), rendered lines: %v", lines)
	}

	// Verify first line of rendered output is line 6 (not line 1-5)
	if len(lines) > 0 {
		firstLine := lines[0]
		if !strings.Contains(firstLine, "line 6") {
			t.Errorf("First visible line should contain 'line 6', got: %q", firstLine)
		}
	}

	// item2 should not be visible yet
	if strings.Contains(rendered, "Item item2") {
		t.Error("item2 should not be visible yet")
	}
}

// TestScrollList_Viewport_ScrollToBottom tests that GotoBottom shows last items correctly
func TestScrollList_Viewport_ScrollToBottom(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create items totaling 25 lines
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
		newMockItem("item4", 5),
		newMockItem("item5", 5),
	}
	sl.SetItems(items)

	// Go to bottom
	sl.GotoBottom()

	rendered := sl.View()

	// Should show last 10 lines (items 4-5)
	if !strings.Contains(rendered, "Item item4 line 1") {
		t.Error("Item 4 should be visible at bottom")
	}
	if !strings.Contains(rendered, "Item item5 line 5") {
		t.Error("Last line of item 5 should be visible")
	}

	// First items should not be visible
	if strings.Contains(rendered, "Item item1") {
		t.Error("Item 1 should not be visible at bottom")
	}
	if strings.Contains(rendered, "Item item2") {
		t.Error("Item 2 should not be visible at bottom")
	}

	// Verify at bottom
	if !sl.AtBottom() {
		t.Error("Should be at bottom after GotoBottom")
	}
}

// TestScrollList_Viewport_SmallViewport tests rendering with very small viewport
func TestScrollList_Viewport_SmallViewport(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 3) // Very small viewport

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	// Should show exactly 3 lines
	if len(lines) > 3 {
		t.Errorf("Rendered %d lines, should not exceed viewport height of 3", len(lines))
	}

	// Only first 3 lines of item1 should be visible
	if !strings.Contains(rendered, "Item item1 line 1") {
		t.Error("Line 1 should be visible")
	}
	if !strings.Contains(rendered, "Item item1 line 3") {
		t.Error("Line 3 should be visible")
	}
	if strings.Contains(rendered, "Item item1 line 4") {
		t.Error("Line 4 should not be visible in small viewport")
	}
}

// TestScrollList_Viewport_ExactFit tests rendering when content exactly fits viewport
func TestScrollList_Viewport_ExactFit(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// Create items totaling exactly 10 lines
	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
	}
	sl.SetItems(items)

	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	// Should show exactly 10 lines
	expectedLines := 10
	if len(lines) != expectedLines {
		t.Errorf("Rendered %d lines, expected %d", len(lines), expectedLines)
	}

	// All content should be visible
	if !strings.Contains(rendered, "Item item1 line 1") {
		t.Error("item1 line 1 should be visible")
	}
	if !strings.Contains(rendered, "Item item2 line 5") {
		t.Error("item2 line 5 should be visible")
	}

	// Should be at both top and bottom
	if sl.currentOffsetInLines() != 0 {
		t.Error("Offset should be 0 when content fits")
	}
	if !sl.AtBottom() {
		t.Error("Should be at bottom when all content visible")
	}
	if sl.ScrollPercent() != 1.0 {
		t.Error("Scroll percent should be 100% when all content visible")
	}
}

// TestScrollList_Viewport_EmptyList tests rendering empty list
func TestScrollList_Viewport_EmptyList(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	// No items added

	rendered := sl.View()

	if rendered != "" {
		t.Error("Empty list should render empty string")
	}

	// Should be at bottom (empty is considered "at bottom")
	if !sl.AtBottom() {
		t.Error("Empty list should report AtBottom=true")
	}
}

// TestScrollList_Viewport_SingleLineItems tests rendering list of single-line items
func TestScrollList_Viewport_SingleLineItems(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 5)

	// Create 10 single-line items (exceeds viewport)
	items := make([]ScrollItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = newMockItem(fmt.Sprintf("item%d", i+1), 1)
	}
	sl.SetItems(items)
	sl.GotoTop()

	rendered := sl.View()
	lines := strings.Split(rendered, "\n")

	// Should show first 5 items
	if len(lines) > 5 {
		t.Errorf("Rendered %d lines, should not exceed viewport height of 5", len(lines))
	}

	// Verify first 5 items visible
	for i := 1; i <= 5; i++ {
		expected := fmt.Sprintf("Item item%d line 1", i)
		if !strings.Contains(rendered, expected) {
			t.Errorf("Item %d should be visible", i)
		}
	}

	// Verify items 6-10 not visible
	for i := 6; i <= 10; i++ {
		unexpected := fmt.Sprintf("Item item%d line 1", i)
		if strings.Contains(rendered, unexpected) {
			t.Errorf("Item %d should not be visible", i)
		}
	}
}

// TestScrollList_Viewport_ScrollByPages tests page up/down scrolling with viewport boundaries
func TestScrollList_Viewport_ScrollByPages(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetFocused(true)

	// Create 30 lines of content
	items := []ScrollItem{
		newMockItem("item1", 10),
		newMockItem("item2", 10),
		newMockItem("item3", 10),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Verify at top
	if sl.currentOffsetInLines() != 0 {
		t.Error("Should start at top")
	}

	// Page down (scroll viewport height = 10 lines)
	sl.Update(tea.KeyPressMsg{Text: "pgdown"})

	if sl.currentOffsetInLines() != 10 {
		t.Errorf("After page down, offset should be 10, got %d", sl.currentOffsetInLines())
	}

	// Page down again
	sl.Update(tea.KeyPressMsg{Text: "pgdown"})

	if sl.currentOffsetInLines() != 20 {
		t.Errorf("After second page down, offset should be 20, got %d", sl.currentOffsetInLines())
	}
	if !sl.AtBottom() {
		t.Error("Should be at bottom after two page downs")
	}

	// Page up (scroll up viewport height)
	sl.Update(tea.KeyPressMsg{Text: "pgup"})

	if sl.currentOffsetInLines() != 10 {
		t.Errorf("After page up, offset should be 10, got %d", sl.currentOffsetInLines())
	}

	// Page up again
	sl.Update(tea.KeyPressMsg{Text: "pgup"})

	if sl.currentOffsetInLines() != 0 {
		t.Errorf("After second page up, offset should be 0, got %d", sl.currentOffsetInLines())
	}
}

// TestScrollList_Viewport_IncrementalScroll tests line-by-line scrolling with viewport boundaries
func TestScrollList_Viewport_IncrementalScroll(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetFocused(true)

	// Create 20 lines of content
	items := []ScrollItem{
		newMockItem("item1", 10),
		newMockItem("item2", 10),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Scroll down one line at a time
	for i := 1; i <= 10; i++ {
		sl.Update(tea.KeyPressMsg{Text: "down"})
		expectedOffset := i
		if sl.currentOffsetInLines() != expectedOffset {
			t.Errorf("After %d down presses, offset should be %d, got %d", i, expectedOffset, sl.currentOffsetInLines())
		}
	}

	// Should be at bottom now (offset 10, showing lines 10-19)
	if !sl.AtBottom() {
		t.Error("Should be at bottom after scrolling down 10 lines")
	}

	// Scroll up one line at a time
	for i := 1; i <= 5; i++ {
		sl.Update(tea.KeyPressMsg{Text: "up"})
		expectedOffset := 10 - i
		if sl.currentOffsetInLines() != expectedOffset {
			t.Errorf("After %d up presses, offset should be %d, got %d", i, expectedOffset, sl.currentOffsetInLines())
		}
	}

	// Should not be at bottom anymore
	if sl.AtBottom() {
		t.Error("Should not be at bottom after scrolling up")
	}
}

// TestScrollList_Viewport_ResizeViewport tests viewport resize handling
func TestScrollList_Viewport_ResizeViewport(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)
	sl.GotoBottom()

	// Verify at bottom with height 10
	if !sl.AtBottom() {
		t.Error("Should be at bottom initially")
	}

	// Increase viewport height to 20 (now everything fits)
	sl.SetHeight(20)

	// Should still be valid, but now showing all content
	if !sl.AtBottom() {
		t.Error("Should be at bottom after increasing height")
	}
	if sl.ScrollPercent() != 1.0 {
		t.Error("Should be 100% when all content fits after resize")
	}

	// Decrease viewport height to 5
	sl.SetHeight(5)

	// Should clamp offset to new bounds
	rendered := sl.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) > 5 {
		t.Errorf("Rendered %d lines, should not exceed new viewport height of 5", len(lines))
	}
}

// TestScrollList_Viewport_ScrollToItemBoundary tests ScrollToItem with viewport boundaries
func TestScrollList_Viewport_ScrollToItemBoundary(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// 6 items, 4 lines each = 24 total lines
	items := []ScrollItem{
		newMockItem("item0", 4),
		newMockItem("item1", 4),
		newMockItem("item2", 4),
		newMockItem("item3", 4),
		newMockItem("item4", 4),
		newMockItem("item5", 4),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Scroll to item 0 (already visible, should not scroll)
	initialOffset := sl.currentOffsetInLines()
	sl.ScrollToItem(0)
	if sl.currentOffsetInLines() != initialOffset {
		t.Error("ScrollToItem(0) should not change offset when item already visible")
	}

	// Scroll to item 5 (starts at line 20, needs scrolling)
	sl.ScrollToItem(5)
	if !sl.AtBottom() {
		t.Error("Scrolling to last item should reach bottom")
	}

	// Verify item 5 is visible
	rendered := sl.View()
	if !strings.Contains(rendered, "Item item5 line 1") {
		t.Error("Item 5 should be visible after ScrollToItem(5)")
	}

	// Scroll to item 2 (middle item, starts at line 8)
	sl.ScrollToItem(2)
	rendered = sl.View()
	if !strings.Contains(rendered, "Item item2 line 1") {
		t.Error("Item 2 should be visible after ScrollToItem(2)")
	}
}

// TestScrollList_Viewport_MultilineItemBoundary tests viewport boundaries with items spanning multiple pages
func TestScrollList_Viewport_MultilineItemBoundary(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	// First item is 25 lines (spans multiple viewport pages)
	items := []ScrollItem{
		newMockItem("huge", 25),
		newMockItem("small", 3),
	}
	sl.SetItems(items)
	sl.GotoTop()

	// Should show first 10 lines of huge item
	rendered := sl.View()
	if !strings.Contains(rendered, "Item huge line 1") {
		t.Error("Line 1 should be visible")
	}
	if strings.Contains(rendered, "Item huge line 11") {
		t.Error("Line 11 should not be visible")
	}

	// Scroll to line 10 (middle of huge item)
	sl.ScrollBy(10)
	rendered = sl.View()
	lines := strings.Split(rendered, "\n")

	if !strings.Contains(rendered, "Item huge line 11") {
		t.Error("Line 11 should be visible after scrolling 10 lines")
	}

	// Check that first rendered line is line 11, not line 1
	if len(lines) > 0 {
		firstLine := lines[0]
		if !strings.Contains(firstLine, "line 11") {
			t.Errorf("First visible line should contain 'line 11', got: %q", firstLine)
		}
	}

	// Scroll to line 20 (near end of huge item)
	sl.ScrollBy(10)
	rendered = sl.View()
	if !strings.Contains(rendered, "Item huge line 21") {
		t.Error("Line 21 should be visible")
	}
	if !strings.Contains(rendered, "Item huge line 25") {
		t.Error("Line 25 (last line of huge) should be visible")
	}
	if !strings.Contains(rendered, "Item small line 1") {
		t.Error("small item should start to be visible")
	}
}

// TestScrollList_Viewport_ViewportHeightChanges tests viewport height changing while scrolled
func TestScrollList_Viewport_ViewportHeightChanges(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)

	items := []ScrollItem{
		newMockItem("item1", 10),
		newMockItem("item2", 10),
		newMockItem("item3", 10),
	}
	sl.SetItems(items)

	// Scroll to middle
	sl.ScrollBy(10)
	middleOffset := sl.currentOffsetInLines()
	if middleOffset != 10 {
		t.Errorf("Expected offset 10, got %d", middleOffset)
	}

	// Increase viewport height
	sl.SetHeight(15)

	// Offset should remain valid (clamped if necessary)
	if sl.currentOffsetInLines() > sl.TotalLineCount()-15 {
		t.Error("Offset should be clamped to valid range after height increase")
	}

	// Decrease viewport height
	sl.SetHeight(5)

	// Offset should still be valid
	rendered := sl.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) > 5 {
		t.Errorf("Rendered %d lines, should not exceed new viewport height of 5", len(lines))
	}
}

// TestScrollList_Viewport_AppendItemWhileScrolled tests appending items while scrolled
func TestScrollList_Viewport_AppendItemWhileScrolled(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetAutoScroll(false) // Disable auto-scroll

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)
	sl.GotoTop()

	initialOffset := sl.currentOffsetInLines()

	// Append new item while at top
	sl.AppendItem(newMockItem("item4", 5))

	// Offset should remain unchanged (not auto-scrolling)
	if sl.currentOffsetInLines() != initialOffset {
		t.Error("Offset should not change when appending with auto-scroll disabled")
	}

	// Verify item 4 exists but is not visible
	if len(sl.items) != 4 {
		t.Errorf("Should have 4 items, got %d", len(sl.items))
	}
	rendered := sl.View()
	if strings.Contains(rendered, "Item item4") {
		t.Error("item4 should not be visible from top of list")
	}
}

// TestScrollList_Viewport_AppendItemWithAutoScroll tests appending items with auto-scroll enabled
func TestScrollList_Viewport_AppendItemWithAutoScroll(t *testing.T) {
	t.Parallel()

	sl := NewScrollList(80, 10)
	sl.SetAutoScroll(true) // Enable auto-scroll

	items := []ScrollItem{
		newMockItem("item1", 5),
		newMockItem("item2", 5),
		newMockItem("item3", 5),
	}
	sl.SetItems(items)

	// Total is 15 lines, viewport is 10, so we're not at bottom initially
	// Need to scroll to bottom first
	sl.GotoBottom()

	// Now should be at bottom
	if !sl.AtBottom() {
		t.Error("Should be at bottom after GotoBottom")
	}

	// Append new item
	sl.AppendItem(newMockItem("item4", 5))

	// Should still be at bottom (auto-scrolled)
	if !sl.AtBottom() {
		t.Error("Should remain at bottom after appending with auto-scroll enabled")
	}

	// Verify item 4 is visible
	rendered := sl.View()
	if !strings.Contains(rendered, "Item item4") {
		t.Error("item4 should be visible at bottom with auto-scroll")
	}
}
