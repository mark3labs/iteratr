package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mark3labs/iteratr/internal/logger"
)

// FileWatcher watches the working directory for filesystem changes using fsnotify.
// It catches all file modifications regardless of source (agent tools, bash, subprocesses).
// Respects .gitignore patterns plus any hard-coded exclusions.
type FileWatcher struct {
	watcher *fsnotify.Watcher
	workDir string
	ignore  *gitIgnore           // gitignore + hard-coded exclusions
	changed map[string]time.Time // relative path -> last event time
	mu      sync.Mutex
	done    chan struct{}
	stopped chan struct{}
}

// NewFileWatcher creates a watcher for the given directory.
// excludeDirs are additional directory names to always exclude (e.g. ".git", data dir).
// .gitignore patterns in workDir are loaded automatically.
func NewFileWatcher(workDir string, excludeDirs []string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Load .gitignore, then add hard-coded exclusions as directory patterns
	ignore := loadGitIgnore(workDir)
	for _, d := range excludeDirs {
		ignore.addPattern(d + "/")
	}

	return &FileWatcher{
		watcher: w,
		workDir: workDir,
		ignore:  ignore,
		changed: make(map[string]time.Time),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}, nil
}

// Start walks the directory tree, adds watches, and starts the event loop.
func (fw *FileWatcher) Start() error {
	if err := fw.addRecursive(fw.workDir); err != nil {
		_ = fw.watcher.Close()
		return err
	}

	go fw.eventLoop()
	logger.Info("FileWatcher started for %s (%d gitignore rules loaded)", fw.workDir, len(fw.ignore.rules))
	return nil
}

// Stop shuts down the watcher and event loop.
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	<-fw.stopped // wait for event loop to exit
	return fw.watcher.Close()
}

// Clear resets the changed file set for a new iteration.
func (fw *FileWatcher) Clear() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.changed = make(map[string]time.Time)
}

// ChangedPaths returns sorted relative paths of files that changed.
func (fw *FileWatcher) ChangedPaths() []string {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	paths := make([]string, 0, len(fw.changed))
	for p := range fw.changed {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// HasChanges returns true if any files have changed since last Clear.
func (fw *FileWatcher) HasChanges() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return len(fw.changed) > 0
}

// addRecursive walks the directory tree and adds watches for all
// non-ignored directories.
func (fw *FileWatcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			return nil
		}

		// Check gitignore rules for this directory
		rel, relErr := filepath.Rel(fw.workDir, path)
		if relErr == nil && rel != "." && fw.ignore.IsIgnored(rel, true) {
			return filepath.SkipDir
		}

		if err := fw.watcher.Add(path); err != nil {
			logger.Warn("FileWatcher: failed to watch %s: %v", path, err)
			if strings.Contains(err.Error(), "no space left on device") ||
				strings.Contains(err.Error(), "too many open files") {
				logger.Error("FileWatcher: inotify watch limit reached. Increase fs.inotify.max_user_watches")
				return filepath.SkipDir
			}
		}
		return nil
	})
}

// eventLoop processes fsnotify events until Stop is called.
func (fw *FileWatcher) eventLoop() {
	defer close(fw.stopped)

	for {
		select {
		case <-fw.done:
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			logger.Warn("FileWatcher error: %v", err)
		}
	}
}

// handleEvent processes a single fsnotify event.
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Only care about Create, Write, Rename
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) && !event.Has(fsnotify.Rename) {
		return
	}

	path := event.Name

	// Normalize to relative path
	relPath, err := filepath.Rel(fw.workDir, path)
	if err != nil {
		relPath = path
	}

	// Skip if it resolves outside working directory
	if strings.HasPrefix(relPath, "..") {
		return
	}

	// Check if a directory — needed for dir-only gitignore rules
	isDir := false
	if info, statErr := os.Stat(path); statErr == nil {
		isDir = info.IsDir()
	}

	// Check gitignore rules
	if fw.ignore.IsIgnored(relPath, isDir) {
		return
	}

	// If a new directory was created, add it to the watch list
	if event.Has(fsnotify.Create) && isDir {
		if err := fw.addRecursive(path); err != nil {
			logger.Warn("FileWatcher: failed to watch new dir %s: %v", path, err)
		}
		return // don't track directory creation itself
	}

	fw.mu.Lock()
	fw.changed[relPath] = time.Now()
	fw.mu.Unlock()
}
