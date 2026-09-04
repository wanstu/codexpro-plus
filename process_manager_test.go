package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMergeEnvironmentOverridesCaseInsensitive(t *testing.T) {
	base := []string{
		"Path=C:\\Windows",
		"CODEXPRO_PORT=1234",
		"OTHER=value",
	}
	got := mergeEnvironment(base, map[string]string{
		"PATH":          `D:\tools\bin`,
		"CODEXPRO_PORT": "8800",
	})
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "Path=C:\\Windows") {
		t.Fatalf("old PATH remained in environment: %s", joined)
	}
	if strings.Contains(joined, "CODEXPRO_PORT=1234") {
		t.Fatalf("old CODEXPRO_PORT remained in environment: %s", joined)
	}
	if !strings.Contains(joined, `PATH=D:\tools\bin`) || !strings.Contains(joined, "CODEXPRO_PORT=8800") {
		t.Fatalf("overrides missing from environment: %s", joined)
	}
	if !strings.Contains(joined, "OTHER=value") {
		t.Fatalf("unrelated environment entry was removed: %s", joined)
	}
}

func TestWaitForPortReleasedWaitsUntilAvailable(t *testing.T) {
	calls := 0
	checker := func(port int) bool {
		calls++
		return calls >= 3
	}
	if err := waitForPortReleased(8800, checker, time.Second); err != nil {
		t.Fatalf("waitForPortReleased() error = %v", err)
	}
	if calls < 3 {
		t.Fatalf("checker calls = %d, want at least 3", calls)
	}
}

func TestWaitForPortReleasedTimesOut(t *testing.T) {
	checker := func(port int) bool { return false }
	started := time.Now()
	err := waitForPortReleased(8800, checker, 150*time.Millisecond)
	if err == nil {
		t.Fatal("waitForPortReleased() error = nil, want timeout")
	}
	if time.Since(started) < 100*time.Millisecond {
		t.Fatalf("waitForPortReleased() returned too early: %s", time.Since(started))
	}
}

func TestRuntimeStatesExposeStartingWorkspace(t *testing.T) {
	manager := NewProcessManager(nil)
	manager.starting["workspace-1"] = true

	states := manager.RuntimeStates()
	if len(states) != 1 || states[0].WorkspaceID != "workspace-1" || !states[0].Starting || states[0].Running {
		t.Fatalf("RuntimeStates() = %#v, want one starting workspace", states)
	}
	if !manager.IsRunning("workspace-1") {
		t.Fatal("IsRunning() should treat starting workspace as active")
	}
}

func TestProcessManagerReadLog(t *testing.T) {
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	service := NewWorkspaceService(store)
	manager := NewProcessManager(service)

	if err := manager.appendLog("workspace-1", "manager: test message"); err != nil {
		t.Fatalf("appendLog() error = %v", err)
	}
	logText, err := manager.ReadLog("workspace-1")
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if !strings.Contains(logText, "manager: test message") {
		t.Fatalf("ReadLog() = %q, want test message", logText)
	}
}

func TestProcessManagerStartRecordsCorePathError(t *testing.T) {
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	service := NewWorkspaceService(store)
	service.portAvailable = func(port int) bool { return true }
	workspace, err := service.AddWorkspace(WorkspaceInput{
		Alias:    "测试项目",
		Path:     t.TempDir(),
		PortMode: "manual",
		Port:     18800,
	})
	if err != nil {
		t.Fatalf("AddWorkspace() error = %v", err)
	}

	manager := NewProcessManager(service)
	manager.portAvailable = func(port int) bool { return true }
	manager.corePathResolver = func() (string, error) {
		return "", errors.New("core intentionally unavailable")
	}

	if _, err := manager.Start(workspace.ID); err == nil || !strings.Contains(err.Error(), "core intentionally unavailable") {
		t.Fatalf("Start() error = %v, want core resolver error", err)
	}
	states := manager.RuntimeStates()
	if len(states) != 1 {
		t.Fatalf("RuntimeStates() len = %d, want 1", len(states))
	}
	if states[0].Running {
		t.Fatal("failed start must not create a running instance")
	}
	if !strings.Contains(states[0].LastError, "core intentionally unavailable") {
		t.Fatalf("LastError = %q", states[0].LastError)
	}
}

func TestProcessManagerRuntimeStateIsNotPersisted(t *testing.T) {
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	service := NewWorkspaceService(store)
	cfg := DefaultConfig()
	cfg.Workspaces = []Workspace{{
		ID:    "workspace-1",
		Alias: "项目",
		Path:  t.TempDir(),
		Port:  8800,
		Token: strings.Repeat("c", tokenBytes*2),
	}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	manager := NewProcessManager(service)
	manager.instances["workspace-1"] = &managedInstance{
		Instance: Instance{
			WorkspaceID: "workspace-1",
			Alias:       "项目",
			Path:        cfg.Workspaces[0].Path,
			PID:         4242,
			Port:        8800,
			Status:      "running",
			StartedAt:   "2026-09-04T20:00:00+08:00",
		},
		done: make(chan struct{}),
	}

	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{"4242", "started_at", "status", "pid"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("runtime field/value %q leaked into config: %s", forbidden, text)
		}
	}
}
