//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	managerRunKeyPath     = `Software\Microsoft\Windows\CurrentVersion\Run`
	legacyManagerRunValue = "CodexProV4"
)

func managerAutoStartEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, managerRunKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取开机自启配置失败: %w", err)
	}
	defer key.Close()

	expected, err := managerAutoStartCommand()
	if err != nil {
		return false, err
	}

	runValue := managerRunValueName()
	value, _, err := key.GetStringValue(runValue)
	if err == nil {
		return strings.EqualFold(strings.TrimSpace(value), expected), nil
	}
	if !errors.Is(err, registry.ErrNotExist) {
		return false, fmt.Errorf("读取开机自启配置失败: %w", err)
	}
	if isDevBuild() {
		return false, nil
	}

	if _, _, err := key.GetStringValue(legacyManagerRunValue); err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("读取旧版开机自启配置失败: %w", err)
	}

	if err := key.SetStringValue(runValue, expected); err != nil {
		return false, fmt.Errorf("迁移开机自启配置失败: %w", err)
	}
	_ = key.DeleteValue(legacyManagerRunValue)
	return true, nil
}

func setManagerAutoStart(enabled bool) error {
	if !enabled {
		key, err := registry.OpenKey(registry.CURRENT_USER, managerRunKeyPath, registry.SET_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("打开开机自启配置失败: %w", err)
		}
		defer key.Close()
		names := []string{managerRunValueName()}
		if !isDevBuild() {
			names = append(names, legacyManagerRunValue)
		}
		for _, name := range names {
			if err := key.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return fmt.Errorf("关闭开机自启失败: %w", err)
			}
		}
		return nil
	}

	command, err := managerAutoStartCommand()
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, managerRunKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("创建开机自启配置失败: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue(managerRunValueName(), command); err != nil {
		return fmt.Errorf("保存开机自启配置失败: %w", err)
	}
	if !isDevBuild() {
		_ = key.DeleteValue(legacyManagerRunValue)
	}
	return nil
}

func managerAutoStartCommand() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取程序路径: %w", err)
	}
	return fmt.Sprintf("\"%s\" --autostart", executable), nil
}
