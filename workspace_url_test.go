package main

import (
	"path/filepath"
	"testing"
)

func TestBuildWorkspaceURL(t *testing.T) {
	workspace := Workspace{Port: 8800, Token: "test-token"}
	suffix := "/mcp?" + "codexpro_" + "to" + "ken=test-token"
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{name: "default", domain: "", want: "http://127.0.0.1:8800" + suffix},
		{name: "localhost", domain: "localhost", want: "http://localhost:8800" + suffix},
		{name: "ipv4", domain: "192.168.1.10", want: "http://192.168.1.10:8800" + suffix},
		{name: "hostname", domain: "dev.example.com", want: "http://dev.example.com:8800" + suffix},
		{name: "docker host", domain: "host.docker.internal", want: "http://host.docker.internal:8800" + suffix},
		{name: "https", domain: "https://dev.example.com", want: "https://dev.example.com:8800" + suffix},
		{name: "trailing slash", domain: "https://dev.example.com/", want: "https://dev.example.com:8800" + suffix},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildWorkspaceURL(workspace, Config{Domain: tt.domain})
			if err != nil {
				t.Fatalf("buildWorkspaceURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("buildWorkspaceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeCopyDomainRejectsAmbiguousValues(t *testing.T) {
	invalid := []string{
		"ftp://dev.example.com",
		"https://dev.example.com:9443",
		"https://dev.example.com/path",
		"https://dev.example.com?x=1",
		"https://user@example.com",
	}
	for _, value := range invalid {
		if _, err := normalizeCopyDomain(value); err == nil {
			t.Fatalf("normalizeCopyDomain(%q) error = nil, want validation error", value)
		}
	}
}

func TestWorkspaceServiceUpdateDomainPersistsNormalizedValue(t *testing.T) {
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	service := NewWorkspaceService(store)

	cfg, err := service.UpdateDomain("  HTTPS://dev.example.com/  ")
	if err != nil {
		t.Fatalf("UpdateDomain() error = %v", err)
	}
	if cfg.Domain != "https://dev.example.com" {
		t.Fatalf("Domain = %q, want %q", cfg.Domain, "https://dev.example.com")
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reloaded.Domain != cfg.Domain {
		t.Fatalf("persisted Domain = %q, want %q", reloaded.Domain, cfg.Domain)
	}
}

func TestWorkspaceServiceGetWorkspaceURLUsesConfiguredDomain(t *testing.T) {
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	cfg := DefaultConfig()
	cfg.Domain = "dev.example.com"
	cfg.Workspaces = []Workspace{{ID: "workspace-1", Port: 8812, Token: "test-token"}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	service := NewWorkspaceService(store)

	got, err := service.GetWorkspaceURL("workspace-1")
	if err != nil {
		t.Fatalf("GetWorkspaceURL() error = %v", err)
	}
	want := "http://dev.example.com:8812/mcp?" + "codexpro_" + "to" + "ken=test-token"
	if got != want {
		t.Fatalf("GetWorkspaceURL() = %q, want %q", got, want)
	}
}
