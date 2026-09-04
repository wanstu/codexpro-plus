package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfigStoreLoadMissingReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	store := NewConfigStore(path)

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("Load() = %#v, want %#v", cfg, DefaultConfig())
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Fatalf("config parent directory was not created: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default config file was not created: %v", err)
	}
}

func TestConfigStoreRoundTripUnicode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewConfigStore(path)
	want := DefaultConfig()
	want.Workspaces = []Workspace{
		{
			ID:        "workspace-1",
			Alias:     "中文项目",
			Path:      `D:\项目\测试`,
			Port:      8801,
			Token:     strings.Repeat("a", tokenBytes*2),
			AutoStart: true,
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "中文项目") {
		t.Fatalf("saved config does not contain UTF-8 alias: %s", raw)
	}
}

func TestConfigStoreMigratesV1WorkspaceToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := map[string]any{
		"version": 1,
		"port_range": map[string]any{
			"start": 8800,
			"end":   8899,
		},
		"workspaces": []map[string]any{
			{
				"id":         "legacy-id",
				"alias":      "旧项目",
				"path":       `D:\legacy`,
				"port":       8803,
				"auto_start": true,
			},
		},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewConfigStore(path)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != configVersion {
		t.Fatalf("Version = %d, want %d", cfg.Version, configVersion)
	}
	if len(cfg.Workspaces) != 1 {
		t.Fatalf("Workspaces len = %d, want 1", len(cfg.Workspaces))
	}
	workspace := cfg.Workspaces[0]
	if workspace.ID != "legacy-id" || workspace.Alias != "旧项目" || workspace.Port != 8803 || !workspace.AutoStart {
		t.Fatalf("legacy workspace fields changed: %#v", workspace)
	}
	if len(workspace.Token) != tokenBytes*2 {
		t.Fatalf("Token length = %d, want %d", len(workspace.Token), tokenBytes*2)
	}
	if workspace.BashMode != defaultBashMode || workspace.WriteMode != defaultWriteMode || workspace.ToolMode != defaultToolMode || !workspace.InheritEnv {
		t.Fatalf("legacy CodexPro settings not migrated: %#v", workspace)
	}

	cfgAgain, err := store.Load()
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if cfgAgain.Workspaces[0].Token != workspace.Token {
		t.Fatal("migration generated a different token on second load")
	}
}

func TestConfigStoreMalformedJSONDoesNotOverwriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	bad := []byte(`{"version":1,"port_range":`)
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	store := NewConfigStore(path)

	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want malformed JSON error")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(raw) != string(bad) {
		t.Fatalf("malformed config was modified: %q", raw)
	}
}

func TestConfigStoreSaveReplacesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewConfigStore(path)

	first := DefaultConfig()
	first.PortRange = PortRange{Start: 8800, End: 8805}
	if err := store.Save(first); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	second := first
	second.PortRange = PortRange{Start: 8900, End: 8910}
	if err := store.Save(second); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.PortRange != second.PortRange {
		t.Fatalf("PortRange = %#v, want %#v", got.PortRange, second.PortRange)
	}
}
