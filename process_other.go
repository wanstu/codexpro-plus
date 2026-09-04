//go:build !windows

package main

import (
	"os"
	"os/exec"
)

func prepareChildCommand(cmd *exec.Cmd) {}

func killProcessTree(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
