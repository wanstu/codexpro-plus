//go:build windows

package main

import (
	"fmt"
	"os"
	"testing"
)

func TestManagerAutoStartCommandUsesCurrentExecutable(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	got, err := managerAutoStartCommand()
	if err != nil {
		t.Fatalf("managerAutoStartCommand() error = %v", err)
	}
	want := fmt.Sprintf("\"%s\" --autostart", executable)
	if got != want {
		t.Fatalf("managerAutoStartCommand() = %q, want %q", got, want)
	}
}
