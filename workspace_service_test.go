package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestWorkspaceService(t *testing.T) (*WorkspaceService, *ConfigStore) {
	t.Helper()
	store := NewConfigStore(filepath.Join(t.TempDir(), "config.json"))
	service := NewWorkspaceService(store)
	service.portAvailable = func(port int) bool { return true }
	return service, store
}

func makeWorkspaceDir(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := ensureTestDir(path); err != nil {
		t.Fatalf("create workspace directory: %v", err)
	}
	return path
}

func ensureTestDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func TestAddWorkspaceAutoPortSkipsConfiguredAndBusy(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.Workspaces = []Workspace{{ID: "existing", Alias: "已有", Path: root, Port: 8800}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	service.portAvailable = func(port int) bool { return port != 8801 }

	workspace, err := service.AddWorkspace(WorkspaceInput{
		Alias:    "新项目",
		Path:     makeWorkspaceDir(t, root, "new"),
		PortMode: "auto",
	})
	if err != nil {
		t.Fatalf("AddWorkspace() error = %v", err)
	}
	if workspace.Port != 8802 {
		t.Fatalf("Port = %d, want 8802", workspace.Port)
	}
	if workspace.ID == "" {
		t.Fatal("workspace ID is empty")
	}
	if len(workspace.Token) != tokenBytes*2 {
		t.Fatalf("Token length = %d, want %d", len(workspace.Token), tokenBytes*2)
	}
	if workspace.BashMode != defaultBashMode || workspace.WriteMode != defaultWriteMode || workspace.ToolMode != defaultToolMode || !workspace.InheritEnv {
		t.Fatalf("default CodexPro settings = %#v", workspace)
	}
}

func TestAddWorkspacePersistsCustomCodexSettings(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	inheritEnv := false
	customToken := strings.Repeat("x", 24)

	workspace, err := service.AddWorkspace(WorkspaceInput{
		Alias:      "受限项目",
		Path:       t.TempDir(),
		PortMode:   "manual",
		Port:       9010,
		Token:      customToken,
		BashMode:   "safe",
		WriteMode:  "handoff",
		ToolMode:   "standard",
		InheritEnv: &inheritEnv,
	})
	if err != nil {
		t.Fatalf("AddWorkspace() error = %v", err)
	}
	if workspace.Token != customToken || workspace.BashMode != "safe" || workspace.WriteMode != "handoff" || workspace.ToolMode != "standard" || workspace.InheritEnv {
		t.Fatalf("custom CodexPro settings = %#v", workspace)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Workspaces) != 1 || cfg.Workspaces[0] != workspace {
		t.Fatalf("persisted Workspace = %#v, want %#v", cfg.Workspaces, workspace)
	}
}

func TestAddWorkspaceRejectsShortToken(t *testing.T) {
	service, _ := newTestWorkspaceService(t)
	_, err := service.AddWorkspace(WorkspaceInput{
		Alias:    "项目",
		Path:     t.TempDir(),
		PortMode: "manual",
		Port:     9011,
		Token:    "too-short",
	})
	if err == nil || !strings.Contains(err.Error(), "至少需要 24 字节") {
		t.Fatalf("short token error = %v", err)
	}
}

func TestAddWorkspaceRejectsDuplicatePath(t *testing.T) {
	service, _ := newTestWorkspaceService(t)
	root := t.TempDir()
	path := makeWorkspaceDir(t, root, "same")

	if _, err := service.AddWorkspace(WorkspaceInput{Alias: "A", Path: path, PortMode: "auto"}); err != nil {
		t.Fatalf("first AddWorkspace() error = %v", err)
	}
	_, err := service.AddWorkspace(WorkspaceInput{Alias: "B", Path: path, PortMode: "auto"})
	if err == nil || !strings.Contains(err.Error(), "已经添加") {
		t.Fatalf("duplicate path error = %v", err)
	}
}

func TestAddWorkspaceRejectsInvalidIdentity(t *testing.T) {
	service, _ := newTestWorkspaceService(t)

	if _, err := service.AddWorkspace(WorkspaceInput{Alias: "  ", Path: t.TempDir(), PortMode: "auto"}); err == nil {
		t.Fatal("empty alias should fail")
	}
	if _, err := service.AddWorkspace(WorkspaceInput{Alias: "项目", Path: "relative/path", PortMode: "auto"}); err == nil {
		t.Fatal("relative path should fail")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := service.AddWorkspace(WorkspaceInput{Alias: "项目", Path: missing, PortMode: "auto"}); err == nil {
		t.Fatal("missing path should fail")
	}
}

func TestAddWorkspaceRejectsManualPortConflict(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.Workspaces = []Workspace{{ID: "existing", Alias: "已有", Path: root, Port: 9000}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := service.AddWorkspace(WorkspaceInput{
		Alias:    "新项目",
		Path:     makeWorkspaceDir(t, root, "new"),
		PortMode: "manual",
		Port:     9000,
	})
	if err == nil || !strings.Contains(err.Error(), "其他工作目录") {
		t.Fatalf("manual conflict error = %v", err)
	}
}

func TestAddWorkspaceRejectsSystemBusyManualPort(t *testing.T) {
	service, _ := newTestWorkspaceService(t)
	service.portAvailable = func(port int) bool { return port != 9001 }

	_, err := service.AddWorkspace(WorkspaceInput{
		Alias:    "项目",
		Path:     t.TempDir(),
		PortMode: "manual",
		Port:     9001,
	})
	if err == nil || !strings.Contains(err.Error(), "其他进程占用") {
		t.Fatalf("busy manual port error = %v", err)
	}
}

func TestUpdateWorkspaceCanKeepOwnPort(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	path := t.TempDir()
	cfg := DefaultConfig()
	originalToken := strings.Repeat("b", tokenBytes*2)
	cfg.Workspaces = []Workspace{{ID: "one", Alias: "旧名称", Path: path, Port: 9002, Token: originalToken}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	service.portAvailable = func(port int) bool { return false }

	workspace, err := service.UpdateWorkspace("one", WorkspaceInput{
		Alias:    "新名称",
		Path:     path,
		PortMode: "manual",
		Port:     9002,
	})
	if err != nil {
		t.Fatalf("UpdateWorkspace() error = %v", err)
	}
	if workspace.Port != 9002 || workspace.Alias != "新名称" {
		t.Fatalf("updated workspace = %#v", workspace)
	}
	if workspace.Token != originalToken {
		t.Fatal("UpdateWorkspace changed the stable token")
	}
}

func TestAllocatePortReportsRangeExhaustion(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	cfg := DefaultConfig()
	cfg.PortRange = PortRange{Start: 8800, End: 8801}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	service.portAvailable = func(port int) bool { return false }

	_, err := service.AddWorkspace(WorkspaceInput{Alias: "项目", Path: t.TempDir(), PortMode: "auto"})
	if err == nil || !strings.Contains(err.Error(), "没有可用端口") {
		t.Fatalf("range exhaustion error = %v", err)
	}
}

func TestUpdatePortRangePreservesExistingWorkspacePort(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	cfg := DefaultConfig()
	cfg.Workspaces = []Workspace{{ID: "one", Alias: "项目", Path: t.TempDir(), Port: 9000}}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	updated, err := service.UpdatePortRange(PortRange{Start: 8800, End: 8805})
	if err != nil {
		t.Fatalf("UpdatePortRange() error = %v", err)
	}
	if updated.Workspaces[0].Port != 9000 {
		t.Fatalf("existing port changed to %d", updated.Workspaces[0].Port)
	}
}

func TestUpdatePortRangeRejectsInvalidRange(t *testing.T) {
	service, _ := newTestWorkspaceService(t)
	if _, err := service.UpdatePortRange(PortRange{Start: 9000, End: 8800}); err == nil {
		t.Fatal("invalid range should fail")
	}
}

func TestDeleteWorkspacePersistsRemoval(t *testing.T) {
	service, store := newTestWorkspaceService(t)
	workspace, err := service.AddWorkspace(WorkspaceInput{Alias: "项目", Path: t.TempDir(), PortMode: "auto"})
	if err != nil {
		t.Fatalf("AddWorkspace() error = %v", err)
	}
	if err := service.DeleteWorkspace(workspace.ID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Workspaces) != 0 {
		t.Fatalf("Workspaces = %#v, want empty", cfg.Workspaces)
	}
}
