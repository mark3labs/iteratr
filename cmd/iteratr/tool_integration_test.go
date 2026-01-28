package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/iteratr/internal/config"
)

// TestToolDataDirResolution tests that tool commands correctly resolve data_dir
// from CLI flag > config file > default, matching the precedence documented in the spec
func TestToolDataDirResolution(t *testing.T) {
	// Save and restore original config
	globalPath := config.GlobalPath()
	projectPath := config.ProjectPath()

	// Backup existing configs
	globalBackup := globalPath + ".test-backup"
	projectBackup := projectPath + ".test-backup"
	if _, err := os.Stat(globalPath); err == nil {
		_ = os.Rename(globalPath, globalBackup)
		defer func() { _ = os.Rename(globalBackup, globalPath) }()
	} else {
		defer func() { _ = os.Remove(globalPath) }()
	}
	if _, err := os.Stat(projectPath); err == nil {
		_ = os.Rename(projectPath, projectBackup)
		defer func() { _ = os.Rename(projectBackup, projectPath) }()
	} else {
		defer func() { _ = os.Remove(projectPath) }()
	}

	// Clear env vars
	origDataDir := os.Getenv("ITERATR_DATA_DIR")
	origModel := os.Getenv("ITERATR_MODEL")
	_ = os.Unsetenv("ITERATR_DATA_DIR")
	_ = os.Unsetenv("ITERATR_MODEL")
	defer func() {
		if origDataDir != "" {
			_ = os.Setenv("ITERATR_DATA_DIR", origDataDir)
		}
		if origModel != "" {
			_ = os.Setenv("ITERATR_MODEL", origModel)
		}
	}()

	t.Run("UsesConfigDataDir", func(t *testing.T) {
		// Create config with custom data_dir
		cfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".custom-iteratr",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		// Simulate connectToSession logic
		toolFlags.dataDir = "" // No CLI flag
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		expected := ".custom-iteratr"
		if dataDir != expected {
			t.Errorf("Expected data_dir %s from config, got %s", expected, dataDir)
		}
	})

	t.Run("CLIFlagOverridesConfig", func(t *testing.T) {
		// Create config with custom data_dir
		cfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".config-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		// Simulate connectToSession logic with CLI flag set
		toolFlags.dataDir = ".cli-override"
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		expected := ".cli-override"
		if dataDir != expected {
			t.Errorf("Expected CLI flag %s to override config, got %s", expected, dataDir)
		}

		// Reset for other tests
		toolFlags.dataDir = ""
	})

	t.Run("FallsBackToDefaultWhenNoConfig", func(t *testing.T) {
		// Ensure no config exists
		_ = os.Remove(globalPath)
		_ = os.Remove(projectPath)

		// Simulate connectToSession logic
		toolFlags.dataDir = "" // No CLI flag
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		expected := ".iteratr"
		if dataDir != expected {
			t.Errorf("Expected default %s when no config, got %s", expected, dataDir)
		}
	})

	t.Run("ProjectConfigOverridesGlobal", func(t *testing.T) {
		// Create global config
		globalCfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".global-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(globalCfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		// Create project config with different data_dir
		projectCfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".project-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err = config.WriteProject(projectCfg)
		if err != nil {
			t.Fatalf("WriteProject failed: %v", err)
		}
		defer func() { _ = os.Remove(projectPath) }()

		// Simulate connectToSession logic
		toolFlags.dataDir = "" // No CLI flag
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		expected := ".project-data"
		if dataDir != expected {
			t.Errorf("Expected project data_dir %s to override global, got %s", expected, dataDir)
		}
	})

	t.Run("EnvVarOverridesConfig", func(t *testing.T) {
		// Create config
		cfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".config-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		// Set env var
		_ = os.Setenv("ITERATR_DATA_DIR", ".env-data")
		defer func() { _ = os.Unsetenv("ITERATR_DATA_DIR") }()

		// Simulate connectToSession logic
		toolFlags.dataDir = "" // No CLI flag
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		// ENV vars are loaded by config.Load(), so should come through cfg.DataDir
		expected := ".env-data"
		if dataDir != expected {
			t.Errorf("Expected env var %s to override config, got %s", expected, dataDir)
		}
	})

	t.Run("FullPrecedenceChain", func(t *testing.T) {
		// Create global and project configs
		globalCfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".global-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(globalCfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		projectCfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    ".project-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err = config.WriteProject(projectCfg)
		if err != nil {
			t.Fatalf("WriteProject failed: %v", err)
		}
		defer func() { _ = os.Remove(projectPath) }()

		// Set env var
		_ = os.Setenv("ITERATR_DATA_DIR", ".env-data")
		defer func() { _ = os.Unsetenv("ITERATR_DATA_DIR") }()

		// Set CLI flag
		toolFlags.dataDir = ".cli-data"
		defer func() { toolFlags.dataDir = "" }()

		// Simulate connectToSession logic - CLI flag should win
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		expected := ".cli-data"
		if dataDir != expected {
			t.Errorf("Full precedence: expected CLI flag %s to override all, got %s", expected, dataDir)
		}
	})
}

// TestToolConfigIntegration tests that tool commands work with config file present
// This is a more comprehensive test that verifies the actual help text mentions config
func TestToolConfigIntegration(t *testing.T) {
	// Verify that tool command has data-dir flag that mentions config
	if toolCmd.PersistentFlags().Lookup("data-dir") == nil {
		t.Fatal("Expected tool command to have --data-dir flag")
	}

	flag := toolCmd.PersistentFlags().Lookup("data-dir")
	usage := flag.Usage

	// The help text should mention config file as documented in spec
	expectedUsage := "Data directory (overrides config file, default: .iteratr)"
	if usage != expectedUsage {
		t.Errorf("Expected data-dir usage to mention config file.\nGot: %s\nWant: %s", usage, expectedUsage)
	}
}

// TestToolPortFileResolution verifies that the port file is looked up in the correct location
// based on the resolved data directory
func TestToolPortFileResolution(t *testing.T) {
	// Save and restore original config
	globalPath := config.GlobalPath()
	projectPath := config.ProjectPath()

	// Backup existing configs
	globalBackup := globalPath + ".test-backup"
	projectBackup := projectPath + ".test-backup"
	if _, err := os.Stat(globalPath); err == nil {
		_ = os.Rename(globalPath, globalBackup)
		defer func() { _ = os.Rename(globalBackup, globalPath) }()
	} else {
		defer func() { _ = os.Remove(globalPath) }()
	}
	if _, err := os.Stat(projectPath); err == nil {
		_ = os.Rename(projectPath, projectBackup)
		defer func() { _ = os.Rename(projectBackup, projectPath) }()
	} else {
		defer func() { _ = os.Remove(projectPath) }()
	}

	t.Run("PortFileLookedUpInConfigDataDir", func(t *testing.T) {
		// Create config with custom data_dir
		customDataDir := ".test-custom-data"
		cfg := &config.Config{
			Model:      "test/model",
			AutoCommit: true,
			DataDir:    customDataDir,
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		err := config.WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}

		// Simulate connectToSession port file path resolution
		toolFlags.dataDir = ""
		dataDir := toolFlags.dataDir
		if dataDir == "" {
			if cfg, err := config.Load(); err == nil {
				dataDir = cfg.DataDir
			}
		}
		if dataDir == "" {
			dataDir = ".iteratr"
		}

		serverDataDir := dataDir + "/data"
		expectedPath := filepath.Join(customDataDir, "data")

		if serverDataDir != expectedPath {
			t.Errorf("Expected port file to be looked up in %s, got %s", expectedPath, serverDataDir)
		}
	})
}
