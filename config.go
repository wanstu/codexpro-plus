package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wanstu/wails-desktop-kit/jsonstore"
	kitpaths "github.com/wanstu/wails-desktop-kit/paths"
)

const (
	configVersion         = 4
	defaultPortStart      = 8800
	defaultPortEnd        = 8899
	tokenBytes            = 32
	defaultBashMode       = "full"
	defaultWriteMode      = "workspace"
	defaultToolMode       = "full"
	legacyConfigDirectory = "codexprov4"
)

type Config struct {
	Version    int         `json:"version"`
	PortRange  PortRange   `json:"port_range"`
	Domain     string      `json:"domain"`
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
	dir, err := kitpaths.ConfigDir(currentConfigDirectory())
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(dir, "config.json"), nil
}

func legacyConfigPath() (string, error) {
	dir, err := kitpaths.ConfigDir(legacyConfigDirectory)
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(dir, "config.json"), nil
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
	if !isDevBuild() {
		if err := migrateLegacyConfigIfNeeded(path); err != nil {
			return nil, err
		}
	}
	return NewConfigStore(path), nil
}

func migrateLegacyConfigIfNeeded(newPath string) error {
	oldPath, err := legacyConfigPath()
	if err != nil {
		return err
	}
	return migrateConfigFileIfNeeded(newPath, oldPath)
}

func migrateConfigFileIfNeeded(newPath, oldPath string) error {
	migrated, err := kitpaths.MigrateFileIfMissing(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("迁移旧版配置失败: %w", err)
	}
	_ = migrated
	return nil
}

func (s *ConfigStore) Path() string {
	return s.path
}

func (s *ConfigStore) values() *jsonstore.Store[Config] {
	return jsonstore.New(s.path, jsonstore.Options[Config]{})
}

func (s *ConfigStore) Load() (Config, error) {
	if s == nil || s.path == "" {
		return Config{}, errors.New("配置文件路径为空")
	}

	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		if err := s.Save(cfg); err != nil {
			return Config{}, fmt.Errorf("创建默认配置失败: %w", err)
		}
		return cfg, nil
	} else if err != nil {
		return Config{}, fmt.Errorf("读取配置失败: %w", err)
	}

	cfg, err := s.values().Load()
	if err != nil {
		return Config{}, fmt.Errorf("读取配置失败: %w", err)
	}

	changed, err := migrateLoadedConfig(&cfg)
	if err != nil {
		return Config{}, err
	}
	if changed {
		if err := s.Save(cfg); err != nil {
			return Config{}, fmt.Errorf("迁移配置失败: %w", err)
		}
	}
	return cfg, nil
}

func migrateLoadedConfig(cfg *Config) (bool, error) {
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
				return false, err
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
	return changed, nil
}

func normalizeConfigForSave(cfg *Config) error {
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
	return nil
}

func (s *ConfigStore) Save(cfg Config) error {
	if s == nil || s.path == "" {
		return errors.New("配置文件路径为空")
	}
	if err := normalizeConfigForSave(&cfg); err != nil {
		return err
	}
	if err := s.values().Save(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
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
