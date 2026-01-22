package tui

import (
	"testing"
)

// TestCalculateLayout_Minimum tests layout at 80x24 (minimum terminal size)
func TestCalculateLayout_Minimum(t *testing.T) {
	width, height := 80, 24
	layout := CalculateLayout(width, height)

	// Should be compact mode
	if layout.Mode != LayoutCompact {
		t.Errorf("Expected LayoutCompact mode at %dx%d, got %v", width, height, layout.Mode)
	}

	// Verify area dimensions
	if layout.Area.Dx() != width || layout.Area.Dy() != height {
		t.Errorf("Area size mismatch: got %dx%d, want %dx%d",
			layout.Area.Dx(), layout.Area.Dy(), width, height)
	}

	// Verify header height
	if layout.Header.Dy() != HeaderHeight {
		t.Errorf("Header height mismatch: got %d, want %d", layout.Header.Dy(), HeaderHeight)
	}

	// Verify status height
	if layout.Status.Dy() != StatusHeight {
		t.Errorf("Status height mismatch: got %d, want %d", layout.Status.Dy(), StatusHeight)
	}

	// Verify footer height
	if layout.Footer.Dy() != FooterHeight {
		t.Errorf("Footer height mismatch: got %d, want %d", layout.Footer.Dy(), FooterHeight)
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
	expectedContentHeight := height - HeaderHeight - StatusHeight - FooterHeight
	if layout.Content.Dy() != expectedContentHeight {
		t.Errorf("Content height mismatch: got %d, want %d",
			layout.Content.Dy(), expectedContentHeight)
	}
}

// TestCalculateLayout_Standard tests layout at 120x40 (standard terminal size)
func TestCalculateLayout_Standard(t *testing.T) {
	width, height := 120, 40
	layout := CalculateLayout(width, height)

	// Should be desktop mode
	if layout.Mode != LayoutDesktop {
		t.Errorf("Expected LayoutDesktop mode at %dx%d, got %v", width, height, layout.Mode)
	}

	// Verify area dimensions
	if layout.Area.Dx() != width || layout.Area.Dy() != height {
		t.Errorf("Area size mismatch: got %dx%d, want %dx%d",
			layout.Area.Dx(), layout.Area.Dy(), width, height)
	}

	// Verify header, status, footer heights
	if layout.Header.Dy() != HeaderHeight {
		t.Errorf("Header height mismatch: got %d, want %d", layout.Header.Dy(), HeaderHeight)
	}
	if layout.Status.Dy() != StatusHeight {
		t.Errorf("Status height mismatch: got %d, want %d", layout.Status.Dy(), StatusHeight)
	}
	if layout.Footer.Dy() != FooterHeight {
		t.Errorf("Footer height mismatch: got %d, want %d", layout.Footer.Dy(), FooterHeight)
	}

	// In desktop mode, sidebar should have width
	if layout.Sidebar.Dx() <= 0 {
		t.Error("Sidebar should have width > 0 in desktop mode")
	}

	// Sidebar width should be reasonable
	if layout.Sidebar.Dx() > SidebarWidthDesktop {
		t.Errorf("Sidebar width %d exceeds maximum %d", layout.Sidebar.Dx(), SidebarWidthDesktop)
	}

	// Main + Sidebar should equal content width
	totalContentWidth := layout.Main.Dx() + layout.Sidebar.Dx()
	if totalContentWidth != layout.Content.Dx() {
		t.Errorf("Main + Sidebar width (%d) doesn't equal content width (%d)",
			totalContentWidth, layout.Content.Dx())
	}

	// Verify content area is properly sized
	expectedContentHeight := height - HeaderHeight - StatusHeight - FooterHeight
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
	layout := CalculateLayout(width, height)

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

	// Main should get remaining width
	expectedMainWidth := layout.Content.Dx() - layout.Sidebar.Dx()
	if layout.Main.Dx() != expectedMainWidth {
		t.Errorf("Main width mismatch: got %d, want %d",
			layout.Main.Dx(), expectedMainWidth)
	}

	// Verify all vertical sections add up to total height
	totalHeight := layout.Header.Dy() + layout.Content.Dy() + layout.Status.Dy() + layout.Footer.Dy()
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
			layout := CalculateLayout(tt.width, tt.height)
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
			layout := CalculateLayout(size.width, size.height)

			// Header should be at top
			if layout.Header.Min.Y != 0 {
				t.Errorf("Header should start at Y=0, got Y=%d", layout.Header.Min.Y)
			}

			// Content should be below header
			if layout.Content.Min.Y != layout.Header.Max.Y {
				t.Errorf("Content should start where header ends")
			}

			// Status should be below content
			if layout.Status.Min.Y != layout.Content.Max.Y {
				t.Errorf("Status should start where content ends")
			}

			// Footer should be below status
			if layout.Footer.Min.Y != layout.Status.Max.Y {
				t.Errorf("Footer should start where status ends")
			}

			// Footer should end at total height
			if layout.Footer.Max.Y != size.height {
				t.Errorf("Footer should end at total height %d, got %d",
					size.height, layout.Footer.Max.Y)
			}

			// In desktop mode, main and sidebar should be side-by-side
			if layout.Mode == LayoutDesktop {
				if layout.Main.Min.X != layout.Content.Min.X {
					t.Error("Main should start at content left edge")
				}
				if layout.Sidebar.Min.X != layout.Main.Max.X {
					t.Error("Sidebar should start where main ends")
				}
				if layout.Sidebar.Max.X != layout.Content.Max.X {
					t.Error("Sidebar should end at content right edge")
				}
			}
		})
	}
}
