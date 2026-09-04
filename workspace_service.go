package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type portAvailabilityFunc func(port int) bool

type WorkspaceService struct {
	store         *ConfigStore
	portAvailable portAvailabilityFunc
	mu            sync.Mutex
}

func NewWorkspaceService(store *ConfigStore) *WorkspaceService {
	return &WorkspaceService{
		store:         store,
		portAvailable: systemPortAvailable,
	}
}

func (s *WorkspaceService) GetConfig() (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loadConfig()
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *WorkspaceService) GetWorkspace(id string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return Workspace{}, errors.New("工作目录 ID 不能为空")
	}
	cfg, err := s.loadConfig()
	if err != nil {
		return Workspace{}, err
	}
	for _, workspace := range cfg.Workspaces {
		if workspace.ID == id {
			return workspace, nil
		}
	}
	return Workspace{}, errors.New("未找到工作目录")
}

func (s *WorkspaceService) AddWorkspace(input WorkspaceInput) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loadConfig()
	if err != nil {
		return Workspace{}, err
	}

	alias, normalizedPath, err := validateWorkspaceIdentity(input.Alias, input.Path)
	if err != nil {
		return Workspace{}, err
	}
	if err := ensureUniquePath(cfg.Workspaces, normalizedPath, ""); err != nil {
		return Workspace{}, err
	}

	port, err := s.resolvePort(cfg, input, "", 0)
	if err != nil {
		return Workspace{}, err
	}

	id, err := newWorkspaceID()
	if err != nil {
		return Workspace{}, err
	}
	token, bashMode, writeMode, toolMode, inheritEnv, err := resolveCodexSettings(input, Workspace{}, true)
	if err != nil {
		return Workspace{}, err
	}

	workspace := Workspace{
		ID:         id,
		Alias:      alias,
		Path:       normalizedPath,
		Port:       port,
		Token:      token,
		BashMode:   bashMode,
		WriteMode:  writeMode,
		ToolMode:   toolMode,
		InheritEnv: inheritEnv,
		AutoStart:  input.AutoStart,
	}
	cfg.Workspaces = append(cfg.Workspaces, workspace)
	if err := s.store.Save(cfg); err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *WorkspaceService) UpdateWorkspace(id string, input WorkspaceInput) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return Workspace{}, errors.New("工作目录 ID 不能为空")
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return Workspace{}, err
	}

	index := -1
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return Workspace{}, errors.New("未找到要编辑的工作目录")
	}

	alias, normalizedPath, err := validateWorkspaceIdentity(input.Alias, input.Path)
	if err != nil {
		return Workspace{}, err
	}
	if err := ensureUniquePath(cfg.Workspaces, normalizedPath, id); err != nil {
		return Workspace{}, err
	}

	current := cfg.Workspaces[index]
	port, err := s.resolvePort(cfg, input, id, current.Port)
	if err != nil {
		return Workspace{}, err
	}
	token, bashMode, writeMode, toolMode, inheritEnv, err := resolveCodexSettings(input, current, false)
	if err != nil {
		return Workspace{}, err
	}

	updated := Workspace{
		ID:         current.ID,
		Alias:      alias,
		Path:       normalizedPath,
		Port:       port,
		Token:      token,
		BashMode:   bashMode,
		WriteMode:  writeMode,
		ToolMode:   toolMode,
		InheritEnv: inheritEnv,
		AutoStart:  input.AutoStart,
	}
	cfg.Workspaces[index] = updated
	if err := s.store.Save(cfg); err != nil {
		return Workspace{}, err
	}
	return updated, nil
}

func (s *WorkspaceService) DeleteWorkspace(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("工作目录 ID 不能为空")
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return err
	}

	index := -1
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("未找到要删除的工作目录")
	}

	cfg.Workspaces = append(cfg.Workspaces[:index], cfg.Workspaces[index+1:]...)
	return s.store.Save(cfg)
}

func (s *WorkspaceService) UpdatePortRange(portRange PortRange) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validatePortRange(portRange); err != nil {
		return Config{}, err
	}
	cfg, err := s.loadConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.PortRange = portRange
	if err := s.store.Save(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *WorkspaceService) loadConfig() (Config, error) {
	if s == nil || s.store == nil {
		return Config{}, errors.New("配置服务未初始化")
	}
	cfg, err := s.store.Load()
	if err != nil {
		return Config{}, err
	}
	if err := validatePortRange(cfg.PortRange); err != nil {
		return Config{}, fmt.Errorf("当前端口范围无效: %w", err)
	}
	return cfg, nil
}

func (s *WorkspaceService) resolvePort(cfg Config, input WorkspaceInput, excludeID string, currentPort int) (int, error) {
	mode := strings.ToLower(strings.TrimSpace(input.PortMode))
	if mode == "" {
		mode = "auto"
	}

	switch mode {
	case "auto":
		return s.allocatePort(cfg, excludeID)
	case "manual":
		if err := validatePort(input.Port); err != nil {
			return 0, err
		}
		if portUsedByWorkspace(cfg.Workspaces, input.Port, excludeID) {
			return 0, fmt.Errorf("端口 %d 已被其他工作目录使用", input.Port)
		}
		if input.Port != currentPort && !s.portChecker()(input.Port) {
			return 0, fmt.Errorf("端口 %d 当前已被系统中的其他进程占用", input.Port)
		}
		return input.Port, nil
	default:
		return 0, errors.New("端口模式必须是 auto 或 manual")
	}
}

func (s *WorkspaceService) allocatePort(cfg Config, excludeID string) (int, error) {
	if err := validatePortRange(cfg.PortRange); err != nil {
		return 0, err
	}
	checker := s.portChecker()
	for port := cfg.PortRange.Start; port <= cfg.PortRange.End; port++ {
		if portUsedByWorkspace(cfg.Workspaces, port, excludeID) {
			continue
		}
		if !checker(port) {
			continue
		}
		return port, nil
	}
	return 0, fmt.Errorf("端口范围 %d-%d 内没有可用端口", cfg.PortRange.Start, cfg.PortRange.End)
}

func (s *WorkspaceService) portChecker() portAvailabilityFunc {
	if s.portAvailable != nil {
		return s.portAvailable
	}
	return systemPortAvailable
}

func validateWorkspaceIdentity(alias, path string) (string, string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", "", errors.New("别名不能为空")
	}

	normalizedPath, err := normalizeWorkspacePath(path)
	if err != nil {
		return "", "", err
	}
	return alias, normalizedPath, nil
}

func normalizeWorkspacePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("工作目录路径不能为空")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("工作目录必须使用绝对路径")
	}

	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("无法规范化工作目录路径: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("工作目录不存在")
		}
		return "", fmt.Errorf("无法访问工作目录: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("工作目录路径不是文件夹")
	}

	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = filepath.Clean(resolved)
	}
	return absolute, nil
}

func ensureUniquePath(workspaces []Workspace, candidate, excludeID string) error {
	for _, workspace := range workspaces {
		if workspace.ID == excludeID {
			continue
		}
		existing := filepath.Clean(strings.TrimSpace(workspace.Path))
		if strings.EqualFold(existing, candidate) {
			return errors.New("该工作目录已经添加过了")
		}
	}
	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("端口必须在 1-65535 之间")
	}
	return nil
}

func validatePortRange(portRange PortRange) error {
	if err := validatePort(portRange.Start); err != nil {
		return fmt.Errorf("起始端口无效: %w", err)
	}
	if err := validatePort(portRange.End); err != nil {
		return fmt.Errorf("结束端口无效: %w", err)
	}
	if portRange.Start > portRange.End {
		return errors.New("起始端口不能大于结束端口")
	}
	return nil
}

func portUsedByWorkspace(workspaces []Workspace, port int, excludeID string) bool {
	for _, workspace := range workspaces {
		if workspace.ID == excludeID {
			continue
		}
		if workspace.Port == port {
			return true
		}
	}
	return false
}

func systemPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func resolveCodexSettings(input WorkspaceInput, current Workspace, isNew bool) (string, string, string, string, bool, error) {
	token := strings.TrimSpace(input.Token)
	if token == "" && !isNew {
		token = current.Token
	}
	if token == "" {
		generated, err := newWorkspaceToken()
		if err != nil {
			return "", "", "", "", false, err
		}
		token = generated
	}
	if len([]byte(token)) < 24 {
		return "", "", "", "", false, errors.New("CodexPro Token 至少需要 24 字节")
	}

	bashMode := strings.ToLower(strings.TrimSpace(input.BashMode))
	if bashMode == "" && !isNew {
		bashMode = strings.ToLower(strings.TrimSpace(current.BashMode))
	}
	if bashMode == "" {
		bashMode = defaultBashMode
	}
	if bashMode != "off" && bashMode != "safe" && bashMode != "full" {
		return "", "", "", "", false, errors.New("Bash 模式必须是 off、safe 或 full")
	}

	writeMode := strings.ToLower(strings.TrimSpace(input.WriteMode))
	if writeMode == "" && !isNew {
		writeMode = strings.ToLower(strings.TrimSpace(current.WriteMode))
	}
	if writeMode == "" {
		writeMode = defaultWriteMode
	}
	if writeMode != "off" && writeMode != "handoff" && writeMode != "workspace" {
		return "", "", "", "", false, errors.New("Write 模式必须是 off、handoff 或 workspace")
	}

	toolMode := strings.ToLower(strings.TrimSpace(input.ToolMode))
	if toolMode == "" && !isNew {
		toolMode = strings.ToLower(strings.TrimSpace(current.ToolMode))
	}
	if toolMode == "" {
		toolMode = defaultToolMode
	}
	if toolMode != "minimal" && toolMode != "standard" && toolMode != "full" {
		return "", "", "", "", false, errors.New("Tool 模式必须是 minimal、standard 或 full")
	}

	inheritEnv := true
	if !isNew {
		inheritEnv = current.InheritEnv
	}
	if input.InheritEnv != nil {
		inheritEnv = *input.InheritEnv
	}
	return token, bashMode, writeMode, toolMode, inheritEnv, nil
}

func newWorkspaceID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成工作目录 ID 失败: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
