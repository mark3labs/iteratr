package tui

import (
	"testing"
)

// TestCalculateLayout_Minimum tests layout at 80x24 (minimum terminal size)
func TestCalculateLayout_Minimum(t *testing.T) {
	width, height := 80, 24
	layout := CalculateLayout(width, height, false)

	// Should be compact mode
	if layout.Mode != LayoutCompact {
		t.Errorf("Expected LayoutCompact mode at %dx%d, got %v", width, height, layout.Mode)
	}

	// Verify area dimensions
	if layout.Area.Dx() != width || layout.Area.Dy() != height {
		t.Errorf("Area size mismatch: got %dx%d, want %dx%d",
			layout.Area.Dx(), layout.Area.Dy(), width, height)
	}

	// Verify status height
	if layout.Status.Dy() != StatusHeight {
		t.Errorf("Status height mismatch: got %d, want %d", layout.Status.Dy(), StatusHeight)
	}

	// In compact mode, sidebar should be empty (no dedicated area)
	if layout.Sidebar.Dx() > 0 || layout.Sidebar.Dy() > 0 {
		t.Errorf("Sidebar should be empty in compact mode, got %dx%d",
			layout.Sidebar.Dx(), layout.Sidebar.Dy())
	}

	// Main should occupy full content width
	if layout.Main.Dx() != width {
		t.Errorf("Main width should equal total width in compact mode: got %d, want %d",
			layout.Main.Dx(), width)
	}

	// Verify content area is properly sized
	expectedContentHeight := height - StatusHeight
	if layout.Content.Dy() != expectedContentHeight {
		t.Errorf("Content height mismatch: got %d, want %d",
			layout.Content.Dy(), expectedContentHeight)
	}
}

// TestCalculateLayout_Standard tests layout at 120x40 (standard terminal size)
func TestCalculateLayout_Standard(t *testing.T) {
	width, height := 120, 40
	layout := CalculateLayout(width, height, false)

	// Should be desktop mode
	if layout.Mode != LayoutDesktop {
		t.Errorf("Expected LayoutDesktop mode at %dx%d, got %v", width, height, layout.Mode)
	}

	// Verify area dimensions
	if layout.Area.Dx() != width || layout.Area.Dy() != height {
		t.Errorf("Area size mismatch: got %dx%d, want %dx%d",
			layout.Area.Dx(), layout.Area.Dy(), width, height)
	}

	// Verify status height
	if layout.Status.Dy() != StatusHeight {
		t.Errorf("Status height mismatch: got %d, want %d", layout.Status.Dy(), StatusHeight)
	}

	// In desktop mode, sidebar should have width
	if layout.Sidebar.Dx() <= 0 {
		t.Error("Sidebar should have width > 0 in desktop mode")
	}

	// Sidebar width should be reasonable
	if layout.Sidebar.Dx() > SidebarWidthDesktop {
		t.Errorf("Sidebar width %d exceeds maximum %d", layout.Sidebar.Dx(), SidebarWidthDesktop)
	}

	// Main + gap (1) + Sidebar should equal content width
	totalContentWidth := layout.Main.Dx() + 1 + layout.Sidebar.Dx()
	if totalContentWidth != layout.Content.Dx() {
		t.Errorf("Main + gap + Sidebar width (%d) doesn't equal content width (%d)",
			totalContentWidth, layout.Content.Dx())
	}

	// Verify content area is properly sized
	expectedContentHeight := height - StatusHeight
	if layout.Content.Dy() != expectedContentHeight {
		t.Errorf("Content height mismatch: got %d, want %d",
			layout.Content.Dy(), expectedContentHeight)
	}

	// Main and Sidebar should have same height as content
	if layout.Main.Dy() != layout.Content.Dy() {
		t.Errorf("Main height (%d) doesn't match content height (%d)",
			layout.Main.Dy(), layout.Content.Dy())
	}
	if layout.Sidebar.Dy() != layout.Content.Dy() {
		t.Errorf("Sidebar height (%d) doesn't match content height (%d)",
			layout.Sidebar.Dy(), layout.Content.Dy())
	}
}

// TestCalculateLayout_Large tests layout at 200x60 (large terminal size)
func TestCalculateLayout_Large(t *testing.T) {
	width, height := 200, 60
	layout := CalculateLayout(width, height, false)

	// Should be desktop mode
	if layout.Mode != LayoutDesktop {
		t.Errorf("Expected LayoutDesktop mode at %dx%d, got %v", width, height, layout.Mode)
	}

	// Verify area dimensions
	if layout.Area.Dx() != width || layout.Area.Dy() != height {
		t.Errorf("Area size mismatch: got %dx%d, want %dx%d",
			layout.Area.Dx(), layout.Area.Dy(), width, height)
	}

	// Sidebar should be at max width (SidebarWidthDesktop)
	// unless content width / 3 is smaller
	maxAllowedSidebarWidth := min(SidebarWidthDesktop, layout.Content.Dx()/3)
	if layout.Sidebar.Dx() != maxAllowedSidebarWidth {
		t.Errorf("Sidebar width mismatch: got %d, want %d",
			layout.Sidebar.Dx(), maxAllowedSidebarWidth)
	}

	// Main should get remaining width minus 1-char gap
	expectedMainWidth := layout.Content.Dx() - layout.Sidebar.Dx() - 1
	if layout.Main.Dx() != expectedMainWidth {
		t.Errorf("Main width mismatch: got %d, want %d",
			layout.Main.Dx(), expectedMainWidth)
	}

	// Verify all vertical sections add up to total height
	totalHeight := layout.Content.Dy() + layout.Status.Dy()
	if totalHeight != height {
		t.Errorf("Vertical sections don't add up: got %d, want %d", totalHeight, height)
	}
}

// TestCalculateLayout_CompactModeTransition tests transition at breakpoints
func TestCalculateLayout_CompactModeTransition(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		wantMode LayoutMode
	}{
		{
			name:     "just below width breakpoint",
			width:    CompactWidthBreakpoint - 1,
			height:   50,
			wantMode: LayoutCompact,
		},
		{
			name:     "just at width breakpoint",
			width:    CompactWidthBreakpoint,
			height:   50,
			wantMode: LayoutDesktop,
		},
		{
			name:     "just below height breakpoint",
			width:    150,
			height:   CompactHeightBreakpoint - 1,
			wantMode: LayoutCompact,
		},
		{
			name:     "just at height breakpoint",
			width:    150,
			height:   CompactHeightBreakpoint,
			wantMode: LayoutDesktop,
		},
		{
			name:     "both at breakpoints",
			width:    CompactWidthBreakpoint,
			height:   CompactHeightBreakpoint,
			wantMode: LayoutDesktop,
		},
		{
			name:     "both below breakpoints",
			width:    CompactWidthBreakpoint - 1,
			height:   CompactHeightBreakpoint - 1,
			wantMode: LayoutCompact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculateLayout(tt.width, tt.height, false)
			if layout.Mode != tt.wantMode {
				t.Errorf("Mode mismatch at %dx%d: got %v, want %v",
					tt.width, tt.height, layout.Mode, tt.wantMode)
			}
		})
	}
}

// TestCalculateLayout_NoOverlaps verifies that layout rectangles don't overlap
func TestCalculateLayout_NoOverlaps(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},
		{120, 40},
		{200, 60},
	}

	for _, size := range sizes {
		t.Run("no overlaps", func(t *testing.T) {
			layout := CalculateLayout(size.width, size.height, false)

			// Content should be at top
			if layout.Content.Min.Y != 0 {
				t.Errorf("Content should start at Y=0, got Y=%d", layout.Content.Min.Y)
			}

			// Status should be below content
			if layout.Status.Min.Y != layout.Content.Max.Y {
				t.Errorf("Status should start where content ends")
			}

			// Status should end at total height
			if layout.Status.Max.Y != size.height {
				t.Errorf("Status should end at total height %d, got %d",
					size.height, layout.Status.Max.Y)
			}

			// In desktop mode, main and sidebar should be side-by-side with 1-char gap
			if layout.Mode == LayoutDesktop {
				if layout.Main.Min.X != layout.Content.Min.X {
					t.Error("Main should start at content left edge")
				}
				if layout.Sidebar.Min.X != layout.Main.Max.X+1 {
					t.Error("Sidebar should start 1 char after main ends (gap)")
				}
				if layout.Sidebar.Max.X != layout.Content.Max.X {
					t.Error("Sidebar should end at content right edge")
				}
			}
		})
	}
}

// TestCalculateLayout_SidebarHidden tests layout with sidebar hidden
func TestCalculateLayout_SidebarHidden(t *testing.T) {
	width, height := 120, 40

	// Test with sidebar visible
	layoutVisible := CalculateLayout(width, height, false)
	if layoutVisible.Mode != LayoutDesktop {
		t.Errorf("Expected LayoutDesktop mode at %dx%d, got %v", width, height, layoutVisible.Mode)
	}
	if layoutVisible.Sidebar.Dx() <= 0 {
		t.Error("Sidebar should have width > 0 when visible")
	}

	// Test with sidebar hidden
	layoutHidden := CalculateLayout(width, height, true)
	if layoutHidden.Mode != LayoutDesktop {
		t.Errorf("Expected LayoutDesktop mode at %dx%d, got %v", width, height, layoutHidden.Mode)
	}

	// When hidden, sidebar should be empty
	if layoutHidden.Sidebar.Dx() > 0 || layoutHidden.Sidebar.Dy() > 0 {
		t.Errorf("Sidebar should be empty when hidden, got %dx%d",
			layoutHidden.Sidebar.Dx(), layoutHidden.Sidebar.Dy())
	}

	// When hidden, main should take full content width
	if layoutHidden.Main.Dx() != width {
		t.Errorf("Main width should equal total width when sidebar hidden: got %d, want %d",
			layoutHidden.Main.Dx(), width)
	}

	// Main should be wider when sidebar is hidden
	if layoutHidden.Main.Dx() <= layoutVisible.Main.Dx() {
		t.Errorf("Main should be wider when sidebar hidden: got %d, want > %d",
			layoutHidden.Main.Dx(), layoutVisible.Main.Dx())
	}
}

// TestCalculateLayout_SidebarHiddenCompactMode tests that sidebar hidden flag
// doesn't affect compact mode (sidebar already hidden)
func TestCalculateLayout_SidebarHiddenCompactMode(t *testing.T) {
	width, height := 80, 24

	// Both should be compact mode and identical
	layoutNotHidden := CalculateLayout(width, height, false)
	layoutHidden := CalculateLayout(width, height, true)

	if layoutNotHidden.Mode != LayoutCompact {
		t.Errorf("Expected LayoutCompact mode at %dx%d", width, height)
	}
	if layoutHidden.Mode != LayoutCompact {
		t.Errorf("Expected LayoutCompact mode at %dx%d", width, height)
	}

	// Both should have empty sidebar
	if layoutNotHidden.Sidebar.Dx() > 0 {
		t.Error("Sidebar should be empty in compact mode (not hidden)")
	}
	if layoutHidden.Sidebar.Dx() > 0 {
		t.Error("Sidebar should be empty in compact mode (hidden)")
	}

	// Both should have same main width
	if layoutNotHidden.Main.Dx() != layoutHidden.Main.Dx() {
		t.Errorf("Main width should be same in compact mode regardless of hidden flag: got %d vs %d",
			layoutNotHidden.Main.Dx(), layoutHidden.Main.Dx())
	}
}
