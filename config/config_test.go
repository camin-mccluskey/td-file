package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"td-file/config"
)

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name           string
		xdgConfigHome  string
		expectedPrefix string
	}{
		{
			name:           "XDG_CONFIG_HOME set",
			xdgConfigHome:  "/custom/config",
			expectedPrefix: "/custom/config/td-file/config.yaml",
		},
		{
			name:           "XDG_CONFIG_HOME not set",
			xdgConfigHome:  "",
			expectedPrefix: filepath.Join(os.Getenv("HOME"), ".config", "td-file", "config.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xdgConfigHome != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
				defer os.Unsetenv("XDG_CONFIG_HOME")
			}

			path, err := config.GetConfigPath()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if path != tt.expectedPrefix {
				t.Errorf("expected path %q, got %q", tt.expectedPrefix, path)
			}
		})
	}
}

func TestLoadConfig_DefaultConfig(t *testing.T) {
	// Set up a temporary home directory
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedBaseDir := filepath.Join(tmpHome, "Documents", "todos")
	if cfg.BaseDir != expectedBaseDir {
		t.Errorf("expected BaseDir to be %q, got %q", expectedBaseDir, cfg.BaseDir)
	}
	if cfg.FilePattern != "todos-{YYYY-MM-DD}.md" {
		t.Errorf("expected FilePattern to be %q, got %q", "todos-{YYYY-MM-DD}.md", cfg.FilePattern)
	}
}

func TestLoadConfig_ExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	// Create config directory and file
	configDir := filepath.Join(tmp, "td-file")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	content := []byte(`file_path: "/tmp/todos.md"`)
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilePath != "/tmp/todos.md" {
		t.Errorf("expected file_path to be /tmp/todos.md, got %q", cfg.FilePath)
	}
}

func TestSaveConfig(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cfg := &config.Config{
		FilePath: "/test/todos.md",
	}

	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify the config was saved correctly
	savedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if savedCfg.FilePath != cfg.FilePath {
		t.Errorf("expected saved file_path to be %q, got %q", cfg.FilePath, savedCfg.FilePath)
	}
}

func TestResolveTodoPath_FilePath(t *testing.T) {
	cfg := &config.Config{FilePath: "/foo/bar.md"}
	path, err := config.ResolveTodoPath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/foo/bar.md" {
		t.Errorf("expected /foo/bar.md, got %q", path)
	}
}

func TestResolveTodoPath_FilePattern(t *testing.T) {
	cfg := &config.Config{
		FilePattern: "todos-{YYYY-MM-DD}.md",
		BaseDir:     "/tmp",
	}
	expected := filepath.Join("/tmp", "todos-"+time.Now().Format("2006-01-02")+".md")
	path, err := config.ResolveTodoPath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestResolveTodoPath_Error(t *testing.T) {
	cfg := &config.Config{}
	_, err := config.ResolveTodoPath(cfg)
	if err == nil {
		t.Error("expected error for missing file_path and file_pattern, got nil")
	}
}
