package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalPath(t *testing.T) {
	tests := []struct {
		name        string
		xdgConfig   string
		wantContain string
	}{
		{
			name:        "with XDG_CONFIG_HOME set",
			xdgConfig:   "/custom/config",
			wantContain: "/custom/config/iteratr/iteratr.yml",
		},
		{
			name:        "without XDG_CONFIG_HOME",
			xdgConfig:   "",
			wantContain: ".config/iteratr/iteratr.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original
			origXDG := os.Getenv("XDG_CONFIG_HOME")
			defer func() {
				if origXDG != "" {
					_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
				} else {
					_ = os.Unsetenv("XDG_CONFIG_HOME")
				}
			}()

			// Set test value
			if tt.xdgConfig != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", tt.xdgConfig)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}

			got := GlobalPath()
			if tt.xdgConfig != "" {
				// When XDG is set, path should be exactly as expected
				if got != tt.wantContain {
					t.Errorf("GlobalPath() = %v, want %v", got, tt.wantContain)
				}
			} else {
				// When XDG not set, should contain .config/iteratr/iteratr.yml
				if !filepath.IsAbs(got) {
					t.Errorf("GlobalPath() should return absolute path, got %v", got)
				}
				if filepath.Base(got) != "iteratr.yml" {
					t.Errorf("GlobalPath() should end with iteratr.yml, got %v", got)
				}
			}
		})
	}
}

func TestProjectPath(t *testing.T) {
	got := ProjectPath()
	want := "iteratr.yml"
	if got != want {
		t.Errorf("ProjectPath() = %v, want %v", got, want)
	}
}

func TestExists(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Save and restore original working directory
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG to temp directory
	xdgDir := filepath.Join(tmpDir, "config")
	_ = os.Setenv("XDG_CONFIG_HOME", xdgDir)

	t.Run("no config exists", func(t *testing.T) {
		if Exists() {
			t.Error("Exists() = true, want false when no config files exist")
		}
	})

	t.Run("global config exists", func(t *testing.T) {
		// Create global config
		globalPath := GlobalPath()
		if err := os.MkdirAll(filepath.Dir(globalPath), 0755); err != nil {
			t.Fatalf("Failed to create global config dir: %v", err)
		}
		if err := os.WriteFile(globalPath, []byte("model: test\n"), 0644); err != nil {
			t.Fatalf("Failed to write global config: %v", err)
		}
		defer func() { _ = os.Remove(globalPath) }()

		if !Exists() {
			t.Error("Exists() = false, want true when global config exists")
		}
	})

	t.Run("project config exists", func(t *testing.T) {
		// Remove global config from previous test
		_ = os.Remove(GlobalPath())

		// Create project config
		projectPath := ProjectPath()
		if err := os.WriteFile(projectPath, []byte("model: test\n"), 0644); err != nil {
			t.Fatalf("Failed to write project config: %v", err)
		}
		defer func() { _ = os.Remove(projectPath) }()

		if !Exists() {
			t.Error("Exists() = false, want true when project config exists")
		}
	})

	t.Run("both configs exist", func(t *testing.T) {
		// Create both configs
		globalPath := GlobalPath()
		if err := os.MkdirAll(filepath.Dir(globalPath), 0755); err != nil {
			t.Fatalf("Failed to create global config dir: %v", err)
		}
		if err := os.WriteFile(globalPath, []byte("model: test\n"), 0644); err != nil {
			t.Fatalf("Failed to write global config: %v", err)
		}
		defer func() { _ = os.Remove(globalPath) }()

		projectPath := ProjectPath()
		if err := os.WriteFile(projectPath, []byte("model: test\n"), 0644); err != nil {
			t.Fatalf("Failed to write project config: %v", err)
		}
		defer func() { _ = os.Remove(projectPath) }()

		if !Exists() {
			t.Error("Exists() = false, want true when both configs exist")
		}
	})
}

func TestWriteGlobal(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Save original XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG to temp directory
	xdgDir := filepath.Join(tmpDir, "config")
	_ = os.Setenv("XDG_CONFIG_HOME", xdgDir)

	cfg := &Config{
		Model:      "test/model",
		AutoCommit: false,
		DataDir:    ".test",
		LogLevel:   "debug",
		LogFile:    "/tmp/test.log",
		Iterations: 5,
		Headless:   true,
		Template:   "custom.txt",
	}

	err := WriteGlobal(cfg)
	if err != nil {
		t.Fatalf("WriteGlobal() error = %v", err)
	}

	// Verify file exists
	globalPath := GlobalPath()
	if _, err := os.Stat(globalPath); err != nil {
		t.Errorf("Config file not created at %s: %v", globalPath, err)
	}

	// Verify file content
	data, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	expectedFields := []string{
		"model: test/model",
		"auto_commit: false",
		"data_dir: .test",
		"log_level: debug",
		"log_file: /tmp/test.log",
		"iterations: 5",
		"headless: true",
		"template: custom.txt",
	}

	for _, field := range expectedFields {
		if !contains(content, field) {
			t.Errorf("Config file missing expected field: %s\nContent:\n%s", field, content)
		}
	}
}

func TestWriteProject(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	cfg := &Config{
		Model:      "project/model",
		AutoCommit: true,
		DataDir:    ".project",
		LogLevel:   "info",
		LogFile:    "",
		Iterations: 0,
		Headless:   false,
		Template:   "",
	}

	err := WriteProject(cfg)
	if err != nil {
		t.Fatalf("WriteProject() error = %v", err)
	}

	// Verify file exists
	projectPath := ProjectPath()
	if _, err := os.Stat(projectPath); err != nil {
		t.Errorf("Config file not created at %s: %v", projectPath, err)
	}

	// Verify file content
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	expectedFields := []string{
		"model: project/model",
		"auto_commit: true",
		"data_dir: .project",
		"log_level: info",
	}

	for _, field := range expectedFields {
		if !contains(content, field) {
			t.Errorf("Config file missing expected field: %s\nContent:\n%s", field, content)
		}
	}
}

func TestLoad_NoConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG to temp directory
	xdgDir := filepath.Join(tmpDir, "config")
	_ = os.Setenv("XDG_CONFIG_HOME", xdgDir)

	// Clear env vars
	origModel := os.Getenv("ITERATR_MODEL")
	defer func() {
		if origModel != "" {
			_ = os.Setenv("ITERATR_MODEL", origModel)
		}
	}()
	_ = os.Unsetenv("ITERATR_MODEL")

	// Load should succeed even without config files (defaults)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults
	if cfg.Model != "" {
		t.Errorf("Load() with no config should have empty model, got %v", cfg.Model)
	}
	if cfg.AutoCommit != true {
		t.Errorf("Load() default AutoCommit = %v, want true", cfg.AutoCommit)
	}
	if cfg.DataDir != ".iteratr" {
		t.Errorf("Load() default DataDir = %v, want .iteratr", cfg.DataDir)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Load() default LogLevel = %v, want info", cfg.LogLevel)
	}
}

func TestLoad_WithGlobalConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG to temp directory
	xdgDir := filepath.Join(tmpDir, "config")
	_ = os.Setenv("XDG_CONFIG_HOME", xdgDir)

	// Clear env vars
	origModel := os.Getenv("ITERATR_MODEL")
	defer func() {
		if origModel != "" {
			_ = os.Setenv("ITERATR_MODEL", origModel)
		}
	}()
	_ = os.Unsetenv("ITERATR_MODEL")

	// Write global config
	globalCfg := &Config{
		Model:      "global/model",
		AutoCommit: false,
		DataDir:    ".global",
		LogLevel:   "warn",
		LogFile:    "",
		Iterations: 3,
		Headless:   false,
		Template:   "",
	}
	if err := WriteGlobal(globalCfg); err != nil {
		t.Fatalf("WriteGlobal() error = %v", err)
	}

	// Load and verify
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Model != globalCfg.Model {
		t.Errorf("Load() Model = %v, want %v", cfg.Model, globalCfg.Model)
	}
	if cfg.AutoCommit != globalCfg.AutoCommit {
		t.Errorf("Load() AutoCommit = %v, want %v", cfg.AutoCommit, globalCfg.AutoCommit)
	}
	if cfg.DataDir != globalCfg.DataDir {
		t.Errorf("Load() DataDir = %v, want %v", cfg.DataDir, globalCfg.DataDir)
	}
	if cfg.LogLevel != globalCfg.LogLevel {
		t.Errorf("Load() LogLevel = %v, want %v", cfg.LogLevel, globalCfg.LogLevel)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with model",
			config: &Config{
				Model:      "anthropic/claude-sonnet-4-5",
				AutoCommit: true,
				DataDir:    ".iteratr",
			},
			wantErr: false,
		},
		{
			name: "invalid config with empty model",
			config: &Config{
				Model:      "",
				AutoCommit: true,
				DataDir:    ".iteratr",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
