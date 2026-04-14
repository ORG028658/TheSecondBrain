package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndAPIKeyLifecycle(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	if !IsFirstRun() {
		t.Fatalf("expected first run before config is saved")
	}

	if err := SaveNew(); err != nil {
		t.Fatalf("SaveNew failed: %v", err)
	}
	if IsFirstRun() {
		t.Fatalf("expected config to exist after SaveNew")
	}

	if _, err := os.Stat(ConfigFilePath()); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	project := filepath.Join(t.TempDir(), "project")
	cfg, err := Load(project)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Paths.Wiki != filepath.Join(project, "wiki") {
		t.Fatalf("unexpected wiki path: %s", cfg.Paths.Wiki)
	}

	if err := UpdateAPIKey("test-key-1234"); err != nil {
		t.Fatalf("UpdateAPIKey failed: %v", err)
	}
	if got := GetAPIKey(); got != "test-key-1234" {
		t.Fatalf("GetAPIKey mismatch: got %q", got)
	}

	if err := Logout(); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if _, err := os.Stat(ConfigDir()); !os.IsNotExist(err) {
		t.Fatalf("expected config dir to be removed, got err=%v", err)
	}
}
