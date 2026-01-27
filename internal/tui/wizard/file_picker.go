package wizard

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// FileItem represents a file or directory in the file picker.
type FileItem struct {
	name  string // Name of file/directory
	path  string // Full path
	isDir bool   // True if directory
}

// ID returns a unique identifier for this item (required by ScrollItem interface).
func (f *FileItem) ID() string {
	return f.path
}

// Render returns the rendered string representation (required by ScrollItem interface).
func (f *FileItem) Render(width int) string {
	icon := "○"
	if f.isDir {
		icon = "▸"
	}

	// Format: "icon name"
	display := icon + " " + f.name

	// Truncate if too long
	if len(display) > width-2 {
		display = display[:width-5] + "..."
	}

	return display
}

// Height returns the number of lines this item occupies (required by ScrollItem interface).
func (f *FileItem) Height() int {
	return 1
}

// FilePickerStep manages the file picker UI step.
type FilePickerStep struct {
	currentPath string          // Current directory path
	items       []*FileItem     // All items in current directory
	scrollList  *tui.ScrollList // Lazy-rendering scroll list for items
	selectedIdx int             // Index of selected item
	width       int             // Available width
	height      int             // Available height
}

// NewFilePickerStep creates a new file picker step.
func NewFilePickerStep() *FilePickerStep {
	// Start in current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	fp := &FilePickerStep{
		currentPath: cwd,
		scrollList:  tui.NewScrollList(60, 10),
		selectedIdx: 0,
		width:       60,
		height:      10,
	}

	// Configure scroll list
	fp.scrollList.SetAutoScroll(false) // Manual navigation for file picker
	fp.scrollList.SetFocused(true)

	// Load initial directory (ignore error, will show empty state)
	_ = fp.loadDirectory(cwd)

	return fp
}

// loadDirectory loads files and directories from the given path.
// Filters to .md, .txt files and directories only.
func (f *FilePickerStep) loadDirectory(path string) error {
	// Read directory entries
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Clear existing items
	f.items = make([]*FileItem, 0)

	// Add parent directory entry if not at root
	absPath, err := filepath.Abs(path)
	if err == nil && absPath != filepath.Dir(absPath) {
		// Not at root, add ".." entry
		parentPath := filepath.Dir(absPath)
		f.items = append(f.items, &FileItem{
			name:  "..",
			path:  parentPath,
			isDir: true,
		})
	}

	// Filter and add entries
	var dirs []*FileItem
	var files []*FileItem

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Include all directories for navigation
			dirs = append(dirs, &FileItem{
				name:  entry.Name(),
				path:  fullPath,
				isDir: true,
			})
		} else {
			// Filter files to .md and .txt only
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".md" || ext == ".txt" {
				files = append(files, &FileItem{
					name:  entry.Name(),
					path:  fullPath,
					isDir: false,
				})
			}
		}
	}

	// Sort directories and files separately
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})

	// Combine: directories first, then files
	f.items = append(f.items, dirs...)
	f.items = append(f.items, files...)

	// Update current path
	f.currentPath = path

	// Reset selection to first item
	f.selectedIdx = 0

	// Update scroll list with new items
	scrollItems := make([]tui.ScrollItem, len(f.items))
	for i, item := range f.items {
		scrollItems[i] = item
	}
	f.scrollList.SetItems(scrollItems)
	f.scrollList.SetSelected(f.selectedIdx)

	return nil
}

// SetSize updates the dimensions for the file picker.
func (f *FilePickerStep) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.scrollList.SetWidth(width)
	f.scrollList.SetHeight(height)
}

// Update handles messages for the file picker step.
func (f *FilePickerStep) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if len(f.items) > 0 && f.selectedIdx > 0 {
				f.selectedIdx--
				f.scrollList.SetSelected(f.selectedIdx)
				f.scrollList.ScrollToItem(f.selectedIdx)
			}
		case "down", "j":
			if len(f.items) > 0 && f.selectedIdx < len(f.items)-1 {
				f.selectedIdx++
				f.scrollList.SetSelected(f.selectedIdx)
				f.scrollList.ScrollToItem(f.selectedIdx)
			}
		case "enter":
			// Handle selection
			if len(f.items) == 0 {
				// No items to select
				return nil
			}
			if f.selectedIdx >= 0 && f.selectedIdx < len(f.items) {
				item := f.items[f.selectedIdx]
				if item.isDir {
					// Navigate into directory (ignore error, UI will show current state)
					_ = f.loadDirectory(item.path)
				} else {
					// File selected - this will be handled by parent wizard
					return func() tea.Msg {
						return FileSelectedMsg{Path: item.path}
					}
				}
			}
		case "backspace":
			// Go up one directory level
			parentPath := filepath.Dir(f.currentPath)
			if parentPath != f.currentPath {
				// Ignore error, UI will maintain current state if navigation fails
				_ = f.loadDirectory(parentPath)
			}
		}
	}

	return nil
}

// View renders the file picker step.
func (f *FilePickerStep) View() string {
	var b strings.Builder

	// Show current path
	t := theme.Current()
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Render(f.currentPath))
	b.WriteString("\n\n")

	// Check if directory is empty (excluding parent directory entry)
	hasFiles := false
	for _, item := range f.items {
		if item.name != ".." {
			hasFiles = true
			break
		}
	}

	if len(f.items) == 0 {
		// Absolutely empty - no parent, no files (shouldn't happen but handle it)
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgOverlay)).Italic(true)
		b.WriteString(emptyStyle.Render("Directory is empty"))
		b.WriteString("\n")
	} else if !hasFiles {
		// Show empty directory message
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgOverlay)).Italic(true)
		b.WriteString(emptyStyle.Render("No .md or .txt files in this directory"))
		b.WriteString("\n\n")

		// Still show parent directory entry if it exists using ScrollList
		if len(f.items) > 0 && f.items[0].name == ".." {
			b.WriteString(f.renderScrollListWithSelection())
		}
	} else {
		// Show items using ScrollList for lazy rendering
		b.WriteString(f.renderScrollListWithSelection())
	}

	// Add spacing before hint bar
	b.WriteString("\n")

	// Hint bar (context-sensitive based on directory state)
	var hintBar string
	if len(f.items) == 0 {
		// No items at all - can only go back or cancel
		hintBar = renderHintBar(
			"backspace", "go up",
			"tab", "buttons",
			"esc", "cancel",
		)
	} else if !hasFiles && len(f.items) > 0 {
		// Empty directory with parent - show simplified hints
		hintBar = renderHintBar(
			"enter/backspace", "go up",
			"tab", "buttons",
			"esc", "cancel",
		)
	} else {
		// Normal hints
		hintBar = renderHintBar(
			"↑↓/j/k", "navigate",
			"enter", "select",
			"tab", "buttons",
			"esc", "back",
		)
	}
	b.WriteString(hintBar)

	return b.String()
}

// renderScrollListWithSelection renders the scroll list.
// ScrollList handles lazy rendering and selection highlighting internally.
func (f *FilePickerStep) renderScrollListWithSelection() string {
	return f.scrollList.View()
}

// SelectedPath returns the currently selected file path (empty if directory selected).
func (f *FilePickerStep) SelectedPath() string {
	if f.selectedIdx >= 0 && f.selectedIdx < len(f.items) {
		item := f.items[f.selectedIdx]
		if !item.isDir {
			return item.path
		}
	}
	return ""
}

// FileSelectedMsg is sent when a file is selected.
type FileSelectedMsg struct {
	Path string
}

// PreferredHeight returns the preferred height for this step's content.
// This allows the modal to size dynamically based on content.
func (f *FilePickerStep) PreferredHeight() int {
	// Fixed overhead:
	// - Current path: 1
	// - Blank line: 1
	// - Blank line before hint bar: 1
	// - Hint bar: 1
	// Total overhead: 4
	overhead := 4

	// File list items (cap at 20 for reasonable modal size)
	listItems := len(f.items)
	if listItems > 20 {
		listItems = 20
	}

	return listItems + overhead
}
