//go:build !windows

package main

import "errors"

func managerAutoStartEnabled() (bool, error) {
	return false, errors.New("开机自启仅支持 Windows")
}

func setManagerAutoStart(enabled bool) error {
	return errors.New("开机自启仅支持 Windows")
}
