package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configVersion      = 3
	defaultPortStart   = 8800
	defaultPortEnd     = 8899
	tokenBytes         = 32
	defaultBashMode    = "full"
	defaultWriteMode   = "workspace"
	defaultToolMode    = "full"
)

type Config struct {
	Version    int         `json:"version"`
	PortRange  PortRange   `json:"port_range"`
	Workspaces []Workspace `json:"workspaces"`
}

type PortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Workspace struct {
	ID         string `json:"id"`
	Alias      string `json:"alias"`
	Path       string `json:"path"`
	Port       int    `json:"port"`
	Token      string `json:"token"`
	BashMode   string `json:"bash_mode"`
	WriteMode  string `json:"write_mode"`
	ToolMode   string `json:"tool_mode"`
	InheritEnv bool   `json:"inherit_env"`
	AutoStart  bool   `json:"auto_start"`
}

type WorkspaceInput struct {
	Alias      string `json:"alias"`
	Path       string `json:"path"`
	PortMode   string `json:"port_mode"`
	Port       int    `json:"port"`
	Token      string `json:"token"`
	BashMode   string `json:"bash_mode"`
	WriteMode  string `json:"write_mode"`
	ToolMode   string `json:"tool_mode"`
	InheritEnv *bool  `json:"inherit_env"`
	AutoStart  bool   `json:"auto_start"`
}

func DefaultConfig() Config {
	return Config{
		Version: configVersion,
		PortRange: PortRange{
			Start: defaultPortStart,
			End:   defaultPortEnd,
		},
		Workspaces: []Workspace{},
	}
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(home, ".config", "codexprov4", "config.json"), nil
}

type ConfigStore struct {
	path string
}

func NewConfigStore(path string) *ConfigStore {
	return &ConfigStore{path: path}
}

func NewDefaultConfigStore() (*ConfigStore, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return NewConfigStore(path), nil
}

func (s *ConfigStore) Path() string {
	return s.path
}

func (s *ConfigStore) Load() (Config, error) {
	if s == nil || s.path == "" {
		return Config{}, errors.New("配置文件路径为空")
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, fmt.Errorf("无法创建配置目录: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		if err := s.Save(cfg); err != nil {
			return Config{}, fmt.Errorf("创建默认配置失败: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("读取配置失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("配置文件格式错误: %w", err)
	}

	changed := false
	legacyBeforeV3 := cfg.Version < 3
	if cfg.Version < configVersion {
		cfg.Version = configVersion
		changed = true
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = []Workspace{}
		changed = true
	}
	for i := range cfg.Workspaces {
		workspace := &cfg.Workspaces[i]
		if workspace.Token == "" {
			token, err := newWorkspaceToken()
			if err != nil {
				return Config{}, err
			}
			workspace.Token = token
			changed = true
		}
		if workspace.BashMode == "" {
			workspace.BashMode = defaultBashMode
			changed = true
		}
		if workspace.WriteMode == "" {
			workspace.WriteMode = defaultWriteMode
			changed = true
		}
		if workspace.ToolMode == "" {
			workspace.ToolMode = defaultToolMode
			changed = true
		}
		if legacyBeforeV3 && !workspace.InheritEnv {
			workspace.InheritEnv = true
			changed = true
		}
	}

	if changed {
		if err := s.Save(cfg); err != nil {
			return Config{}, fmt.Errorf("迁移配置失败: %w", err)
		}
	}
	return cfg, nil
}

func (s *ConfigStore) Save(cfg Config) error {
	if s == nil || s.path == "" {
		return errors.New("配置文件路径为空")
	}

	cfg.Version = configVersion
	if cfg.Workspaces == nil {
		cfg.Workspaces = []Workspace{}
	}
	for i := range cfg.Workspaces {
		workspace := &cfg.Workspaces[i]
		if workspace.Token == "" {
			token, err := newWorkspaceToken()
			if err != nil {
				return err
			}
			workspace.Token = token
		}
		if workspace.BashMode == "" {
			workspace.BashMode = defaultBashMode
		}
		if workspace.WriteMode == "" {
			workspace.WriteMode = defaultWriteMode
		}
		if workspace.ToolMode == "" {
			workspace.ToolMode = defaultToolMode
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("无法创建配置目录: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时配置失败: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("设置临时配置权限失败: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("写入临时配置失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("同步临时配置失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("关闭临时配置失败: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("替换配置文件失败: %w", err)
	}
	return nil
}

func newWorkspaceToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成工作目录 token 失败: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
