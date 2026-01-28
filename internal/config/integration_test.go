package config

import (
	"os"
	"testing"
)

// TestE2EConfigFlow tests the end-to-end flow of setup creating config and build reading it
func TestE2EConfigFlow(t *testing.T) {
	// Save and restore original config
	globalPath := GlobalPath()
	projectPath := ProjectPath()
	
	// Backup existing configs
	globalBackup := globalPath + ".test-backup"
	projectBackup := projectPath + ".test-backup"
	if _, err := os.Stat(globalPath); err == nil {
		os.Rename(globalPath, globalBackup)
		defer os.Rename(globalBackup, globalPath)
	} else {
		defer os.Remove(globalPath)
	}
	if _, err := os.Stat(projectPath); err == nil {
		os.Rename(projectPath, projectBackup)
		defer os.Rename(projectBackup, projectPath)
	} else {
		defer os.Remove(projectPath)
	}
	
	// Clear env vars
	origModel := os.Getenv("ITERATR_MODEL")
	origAutoCommit := os.Getenv("ITERATR_AUTO_COMMIT")
	origDataDir := os.Getenv("ITERATR_DATA_DIR")
	os.Unsetenv("ITERATR_MODEL")
	os.Unsetenv("ITERATR_AUTO_COMMIT")
	os.Unsetenv("ITERATR_DATA_DIR")
	defer func() {
		if origModel != "" {
			os.Setenv("ITERATR_MODEL", origModel)
		}
		if origAutoCommit != "" {
			os.Setenv("ITERATR_AUTO_COMMIT", origAutoCommit)
		}
		if origDataDir != "" {
			os.Setenv("ITERATR_DATA_DIR", origDataDir)
		}
	}()
	
	t.Run("SetupCreatesGlobalConfig", func(t *testing.T) {
		// Simulate setup command creating config
		cfg := &Config{
			Model:      "anthropic/claude-sonnet-4-5",
			AutoCommit: true,
			DataDir:    ".iteratr",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		
		err := WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}
		
		// Verify file exists
		if !Exists() {
			t.Fatal("Config file should exist after WriteGlobal")
		}
		
		// Load and verify
		loaded, err := Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if loaded.Model != cfg.Model {
			t.Errorf("Expected model %s, got %s", cfg.Model, loaded.Model)
		}
		if loaded.AutoCommit != cfg.AutoCommit {
			t.Errorf("Expected AutoCommit %v, got %v", cfg.AutoCommit, loaded.AutoCommit)
		}
		if loaded.DataDir != cfg.DataDir {
			t.Errorf("Expected DataDir %s, got %s", cfg.DataDir, loaded.DataDir)
		}
	})
	
	t.Run("ProjectConfigOverridesGlobal", func(t *testing.T) {
		// Create global config
		globalCfg := &Config{
			Model:      "global/model",
			AutoCommit: true,
			DataDir:    ".global-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 5,
			Headless:   false,
			Template:   "",
		}
		err := WriteGlobal(globalCfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}
		
		// Create project config with different values
		projectCfg := &Config{
			Model:      "project/model",
			AutoCommit: false,
			DataDir:    ".project-data",
			LogLevel:   "debug",
			LogFile:    "",
			Iterations: 10,
			Headless:   true,
			Template:   "custom.txt",
		}
		err = WriteProject(projectCfg)
		if err != nil {
			t.Fatalf("WriteProject failed: %v", err)
		}
		defer os.Remove(projectPath)
		
		// Load - should get project values
		loaded, err := Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if loaded.Model != projectCfg.Model {
			t.Errorf("Expected project model %s, got %s", projectCfg.Model, loaded.Model)
		}
		if loaded.AutoCommit != projectCfg.AutoCommit {
			t.Errorf("Expected project AutoCommit %v, got %v", projectCfg.AutoCommit, loaded.AutoCommit)
		}
		if loaded.DataDir != projectCfg.DataDir {
			t.Errorf("Expected project DataDir %s, got %s", projectCfg.DataDir, loaded.DataDir)
		}
		if loaded.LogLevel != projectCfg.LogLevel {
			t.Errorf("Expected project LogLevel %s, got %s", projectCfg.LogLevel, loaded.LogLevel)
		}
	})
	
	t.Run("EnvVarOverridesConfig", func(t *testing.T) {
		// Create config
		cfg := &Config{
			Model:      "config/model",
			AutoCommit: true,
			DataDir:    ".config-data",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 5,
			Headless:   false,
			Template:   "",
		}
		err := WriteGlobal(cfg)
		if err != nil {
			t.Fatalf("WriteGlobal failed: %v", err)
		}
		
		// Set env vars
		os.Setenv("ITERATR_MODEL", "env/model")
		os.Setenv("ITERATR_AUTO_COMMIT", "false")
		os.Setenv("ITERATR_DATA_DIR", ".env-data")
		defer func() {
			os.Unsetenv("ITERATR_MODEL")
			os.Unsetenv("ITERATR_AUTO_COMMIT")
			os.Unsetenv("ITERATR_DATA_DIR")
		}()
		
		// Load - should get env values
		loaded, err := Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if loaded.Model != "env/model" {
			t.Errorf("Expected env model, got %s", loaded.Model)
		}
		if loaded.AutoCommit != false {
			t.Errorf("Expected env AutoCommit false, got %v", loaded.AutoCommit)
		}
		if loaded.DataDir != ".env-data" {
			t.Errorf("Expected env DataDir, got %s", loaded.DataDir)
		}
	})
	
	t.Run("ValidateRejectsEmptyModel", func(t *testing.T) {
		cfg := &Config{
			Model:      "",
			AutoCommit: true,
			DataDir:    ".iteratr",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("Expected Validate to reject empty model")
		}
	})
	
	t.Run("ValidateAcceptsNonEmptyModel", func(t *testing.T) {
		cfg := &Config{
			Model:      "anthropic/claude-sonnet-4-5",
			AutoCommit: true,
			DataDir:    ".iteratr",
			LogLevel:   "info",
			LogFile:    "",
			Iterations: 0,
			Headless:   false,
			Template:   "",
		}
		
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected Validate to accept non-empty model, got: %v", err)
		}
	})
}
