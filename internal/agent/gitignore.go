package agent

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitIgnore matches file paths against .gitignore-style patterns.
// Supports: globs (*.log), directory-only (build/), root-relative (/dist),
// double-star (**/foo, foo/**), negation (!important.log), comments (#).
type gitIgnore struct {
	rules []ignoreRule
}

type ignoreRule struct {
	pattern  string // cleaned pattern (no leading /, no trailing /)
	negate   bool   // ! prefix — un-ignores a previously matched path
	dirOnly  bool   // trailing / — only matches directories
	anchored bool   // contains / (not just trailing) — match against full path
}

// loadGitIgnore parses the .gitignore in workDir.
// Returns an empty (non-nil) matcher if the file doesn't exist.
func loadGitIgnore(workDir string) *gitIgnore {
	gi := &gitIgnore{}
	gi.loadFile(filepath.Join(workDir, ".gitignore"))
	return gi
}

// loadFile parses a single .gitignore file and appends its rules.
func (gi *gitIgnore) loadFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if rule, ok := parseIgnoreRule(scanner.Text()); ok {
			gi.rules = append(gi.rules, rule)
		}
	}
}

// addPattern adds a programmatic pattern (for hard-coded exclusions like .git).
func (gi *gitIgnore) addPattern(pattern string) {
	if rule, ok := parseIgnoreRule(pattern); ok {
		gi.rules = append(gi.rules, rule)
	}
}

// IsIgnored returns true if relPath should be excluded.
// isDir should be true when the path is known to be a directory.
// Rules are evaluated in order; the last matching rule wins.
// A path is also ignored if any of its parent directories are ignored.
func (gi *gitIgnore) IsIgnored(relPath string, isDir bool) bool {
	if len(gi.rules) == 0 {
		return false
	}

	// Check if any parent directory is ignored (e.g. .git/ ignores .git/HEAD)
	parts := strings.Split(relPath, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		if gi.matchPath(parent, true) {
			return true
		}
	}

	return gi.matchPath(relPath, isDir)
}

// matchPath checks the rules against a single path (no parent traversal).
func (gi *gitIgnore) matchPath(relPath string, isDir bool) bool {
	ignored := false
	for _, rule := range gi.rules {
		if rule.dirOnly && !isDir {
			continue
		}
		if matchRule(rule, relPath) {
			ignored = !rule.negate
		}
	}
	return ignored
}

func parseIgnoreRule(line string) (ignoreRule, bool) {
	// Trim trailing whitespace
	line = strings.TrimRight(line, " \t")

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreRule{}, false
	}

	var r ignoreRule

	// Negation
	if strings.HasPrefix(line, "!") {
		r.negate = true
		line = line[1:]
	}

	// Directory-only (trailing /)
	if strings.HasSuffix(line, "/") {
		r.dirOnly = true
		line = strings.TrimRight(line, "/")
	}

	// Anchoring: leading / means root-relative; a / anywhere in the middle
	// also anchors the pattern (match against full path, not just basename).
	if strings.HasPrefix(line, "/") {
		r.anchored = true
		line = line[1:]
	} else if strings.Contains(line, "/") {
		// Contains / in the middle → anchored (unless it's just **/...)
		if !strings.HasPrefix(line, "**/") {
			r.anchored = true
		}
	}

	r.pattern = line
	if line == "" {
		return ignoreRule{}, false
	}
	return r, true
}

// matchRule checks whether a single rule matches a relative path.
func matchRule(rule ignoreRule, relPath string) bool {
	pattern := rule.pattern

	// Handle leading **/ — match at any depth (like non-anchored)
	if strings.HasPrefix(pattern, "**/") {
		rest := pattern[3:]
		return matchAnySuffix(rest, relPath)
	}

	// Handle trailing /** — match everything under prefix
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return relPath == prefix || strings.HasPrefix(relPath, prefix+"/")
	}

	// Handle **/ in the middle: prefix/**/suffix
	if idx := strings.Index(pattern, "/**/"); idx >= 0 {
		prefix := pattern[:idx]
		suffix := pattern[idx+4:]
		if relPath != prefix && !strings.HasPrefix(relPath, prefix+"/") {
			return false
		}
		remaining := strings.TrimPrefix(relPath, prefix+"/")
		return matchAnySuffix(suffix, remaining)
	}

	if rule.anchored {
		return matchGlob(pattern, relPath)
	}

	// Non-anchored: match against the basename
	return matchGlob(pattern, filepath.Base(relPath))
}

// matchAnySuffix tries to match pattern against relPath or any suffix
// starting after a path separator.
func matchAnySuffix(pattern, relPath string) bool {
	if matchGlob(pattern, relPath) {
		return true
	}
	for i := 0; i < len(relPath); i++ {
		if relPath[i] == '/' {
			if matchGlob(pattern, relPath[i+1:]) {
				return true
			}
		}
	}
	return false
}

// matchGlob wraps filepath.Match, returning false on malformed patterns.
func matchGlob(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)
	return matched
}
