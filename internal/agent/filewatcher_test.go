package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWatcher(dir, []string{".git", "node_modules"})
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer func() { _ = fw.watcher.Close() }()

	if fw.workDir != dir {
		t.Errorf("workDir = %q, want %q", fw.workDir, dir)
	}
	if fw.ignore == nil {
		t.Error("expected ignore to be initialized")
	}
	// Hard-coded excludes are loaded as dir-only rules
	if !fw.ignore.IsIgnored(".git", true) {
		t.Error("expected .git to be excluded")
	}
	if !fw.ignore.IsIgnored("node_modules", true) {
		t.Error("expected node_modules to be excluded")
	}
}

func TestFileWatcher_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWatcher(dir, []string{".git"})
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create a file
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Wait for fsnotify to pick it up
	time.Sleep(200 * time.Millisecond)

	if !fw.HasChanges() {
		t.Error("expected HasChanges() = true after file creation")
	}

	paths := fw.ChangedPaths()
	if len(paths) != 1 || paths[0] != "test.txt" {
		t.Errorf("ChangedPaths() = %v, want [test.txt]", paths)
	}
}

func TestFileWatcher_DetectsModifiedFile(t *testing.T) {
	dir := t.TempDir()

	// Create file before starting watcher
	filePath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(filePath, []byte("original"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	fw, err := NewFileWatcher(dir, nil)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Modify the file
	if err := os.WriteFile(filePath, []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if !fw.HasChanges() {
		t.Error("expected HasChanges() = true after file modification")
	}

	paths := fw.ChangedPaths()
	if len(paths) != 1 || paths[0] != "existing.txt" {
		t.Errorf("ChangedPaths() = %v, want [existing.txt]", paths)
	}
}

func TestFileWatcher_ExcludesDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create excluded directory before starting watcher
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	fw, err := NewFileWatcher(dir, []string{".git"})
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create file in excluded directory
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Also create a tracked file for comparison
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	paths := fw.ChangedPaths()
	for _, p := range paths {
		if filepath.Dir(p) == ".git" || p == ".git" {
			t.Errorf("excluded .git path found in changes: %s", p)
		}
	}

	found := false
	for _, p := range paths {
		if p == "tracked.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tracked.txt in changes, got %v", paths)
	}
}

func TestFileWatcher_ExcludesGitignorePatterns(t *testing.T) {
	dir := t.TempDir()

	// Write a .gitignore
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\nbuild/\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create build directory
	if err := os.MkdirAll(filepath.Join(dir, "build"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	fw, err := NewFileWatcher(dir, []string{".git"})
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create a .log file (gitignored)
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create a file in build/ (gitignored directory)
	if err := os.WriteFile(filepath.Join(dir, "build", "output.bin"), []byte("bin"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create a tracked file
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	paths := fw.ChangedPaths()

	// Should NOT contain the .log or build/ files
	for _, p := range paths {
		if filepath.Ext(p) == ".log" {
			t.Errorf("gitignored .log file found in changes: %s", p)
		}
		if strings.HasPrefix(p, "build") {
			t.Errorf("gitignored build/ file found in changes: %s", p)
		}
	}

	// Should contain main.go (and possibly .gitignore itself)
	found := false
	for _, p := range paths {
		if p == "main.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected main.go in changes, got %v", paths)
	}
}

func TestFileWatcher_DetectsSubdirectoryFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a subdirectory before starting
	subDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	fw, err := NewFileWatcher(dir, nil)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create file in subdirectory
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	paths := fw.ChangedPaths()
	expected := filepath.Join("src", "main.go")
	found := false
	for _, p := range paths {
		if p == expected {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s in changes, got %v", expected, paths)
	}
}

func TestFileWatcher_DetectsNewSubdirectoryFiles(t *testing.T) {
	dir := t.TempDir()

	fw, err := NewFileWatcher(dir, nil)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create a new subdirectory AFTER watcher started
	newDir := filepath.Join(dir, "internal")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Give watcher time to register the new directory
	time.Sleep(200 * time.Millisecond)

	// Create file in the new subdirectory
	if err := os.WriteFile(filepath.Join(newDir, "handler.go"), []byte("package internal"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	paths := fw.ChangedPaths()
	expected := filepath.Join("internal", "handler.go")
	found := false
	for _, p := range paths {
		if p == expected {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s in changes, got %v", expected, paths)
	}
}

func TestFileWatcher_Clear(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWatcher(dir, nil)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Create a file
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if !fw.HasChanges() {
		t.Error("expected changes before clear")
	}

	fw.Clear()

	if fw.HasChanges() {
		t.Error("expected no changes after clear")
	}
	if len(fw.ChangedPaths()) != 0 {
		t.Error("expected empty ChangedPaths after clear")
	}
}

func TestFileWatcher_DataDirExclusion(t *testing.T) {
	dir := t.TempDir()

	// Create .iteratr directory
	dataDir := filepath.Join(dir, ".iteratr")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Data dir is always excluded from watching (commit_data_dir only
	// controls whether the auto-commit prompt includes it)
	fw, err := NewFileWatcher(dir, []string{".git", ".iteratr"})
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = fw.Stop() }()

	// Write to data dir — should be ignored
	if err := os.WriteFile(filepath.Join(dataDir, "state.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write to tracked area
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	paths := fw.ChangedPaths()
	for _, p := range paths {
		if strings.HasPrefix(p, ".iteratr") {
			t.Errorf("excluded .iteratr path found in changes: %s", p)
		}
	}

	found := false
	for _, p := range paths {
		if p == "main.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected main.go in changes, got %v", paths)
	}
}

// Test MergeWatcherPaths on FileTracker

func TestMergeWatcherPaths_AddsNewPaths(t *testing.T) {
	ft := NewFileTracker("/home/user/project")

	ft.MergeWatcherPaths([]string{"file1.go", "file2.go"})

	if ft.Count() != 2 {
		t.Errorf("Count() = %d, want 2", ft.Count())
	}

	paths := ft.ModifiedPaths()
	sort.Strings(paths)
	if len(paths) != 2 || paths[0] != "file1.go" || paths[1] != "file2.go" {
		t.Errorf("ModifiedPaths() = %v, want [file1.go file2.go]", paths)
	}
}

func TestMergeWatcherPaths_PreservesACPMetadata(t *testing.T) {
	ft := NewFileTracker("/home/user/project")

	// ACP event provides rich metadata
	ft.RecordChange("/home/user/project/main.go", false, 15, 3)

	// Watcher also sees the same file — should NOT overwrite
	ft.MergeWatcherPaths([]string{"main.go"})

	if ft.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (no duplicate)", ft.Count())
	}

	change := ft.Get("main.go")
	if change == nil {
		t.Fatal("expected change for main.go")
	}
	if change.Additions != 15 || change.Deletions != 3 {
		t.Errorf("ACP metadata overwritten: additions=%d deletions=%d, want 15/3",
			change.Additions, change.Deletions)
	}
}

func TestMergeWatcherPaths_MixedSources(t *testing.T) {
	ft := NewFileTracker("/home/user/project")

	// ACP tracks one file with metadata
	ft.RecordChange("/home/user/project/api.go", true, 50, 0)

	// Watcher sees api.go (already tracked) + util.go (new, via bash)
	ft.MergeWatcherPaths([]string{"api.go", "util.go"})

	if ft.Count() != 2 {
		t.Errorf("Count() = %d, want 2", ft.Count())
	}

	// api.go should retain ACP metadata
	apiChange := ft.Get("api.go")
	if apiChange == nil {
		t.Fatal("expected change for api.go")
	}
	if !apiChange.IsNew || apiChange.Additions != 50 {
		t.Errorf("api.go metadata wrong: isNew=%v additions=%d", apiChange.IsNew, apiChange.Additions)
	}

	// util.go should have minimal metadata
	utilChange := ft.Get("util.go")
	if utilChange == nil {
		t.Fatal("expected change for util.go")
	}
	if utilChange.IsNew || utilChange.Additions != 0 || utilChange.Deletions != 0 {
		t.Errorf("util.go should have minimal metadata: isNew=%v additions=%d deletions=%d",
			utilChange.IsNew, utilChange.Additions, utilChange.Deletions)
	}
	if utilChange.AbsPath != filepath.Join("/home/user/project", "util.go") {
		t.Errorf("util.go AbsPath = %q, want joined path", utilChange.AbsPath)
	}
}

func TestMergeWatcherPaths_EmptyList(t *testing.T) {
	ft := NewFileTracker("/home/user/project")
	ft.RecordChange("/home/user/project/main.go", false, 1, 1)

	ft.MergeWatcherPaths(nil)
	ft.MergeWatcherPaths([]string{})

	if ft.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (unchanged)", ft.Count())
	}
}
