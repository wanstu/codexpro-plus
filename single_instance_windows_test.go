//go:build windows

package main

import (
	"path/filepath"
	"testing"
)

func TestAcquireInstanceLockAllowsOnlyOneOwner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), managerLockFileName)

	releaseFirst, primary, err := acquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("first acquireInstanceLock() error = %v", err)
	}
	if !primary {
		t.Fatal("first acquireInstanceLock() primary = false, want true")
	}
	defer releaseFirst()

	releaseSecond, primary, err := acquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("second acquireInstanceLock() error = %v", err)
	}
	defer releaseSecond()
	if primary {
		t.Fatal("second acquireInstanceLock() primary = true, want false")
	}

	releaseFirst()
	releaseThird, primary, err := acquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("third acquireInstanceLock() error = %v", err)
	}
	defer releaseThird()
	if !primary {
		t.Fatal("third acquireInstanceLock() primary = false after release, want true")
	}
}

func TestDevSingleInstanceControlDirUsesDevConfig(t *testing.T) {
	withBuildProfile(t, devBuildProfile)
	dir, err := singleInstanceControlDir()
	if err != nil {
		t.Fatalf("singleInstanceControlDir() error = %v", err)
	}
	if filepath.Base(dir) != devConfigDirectory {
		t.Fatalf("singleInstanceControlDir() = %q, want base %q", dir, devConfigDirectory)
	}
}

func TestWakeRequestLifecycle(t *testing.T) {
	wakePath := filepath.Join(t.TempDir(), managerWakeFileName)

	if err := prepareWakePath(wakePath); err != nil {
		t.Fatalf("prepareWakePath() error = %v", err)
	}
	requested, err := consumeWakePath(wakePath)
	if err != nil {
		t.Fatalf("consumeWakePath() before request error = %v", err)
	}
	if requested {
		t.Fatal("consumeWakePath() before request = true, want false")
	}

	if err := requestWakePath(wakePath); err != nil {
		t.Fatalf("requestWakePath() error = %v", err)
	}
	requested, err = consumeWakePath(wakePath)
	if err != nil {
		t.Fatalf("consumeWakePath() after request error = %v", err)
	}
	if !requested {
		t.Fatal("consumeWakePath() after request = false, want true")
	}

	requested, err = consumeWakePath(wakePath)
	if err != nil {
		t.Fatalf("consumeWakePath() second consume error = %v", err)
	}
	if requested {
		t.Fatal("consumeWakePath() second consume = true, want false")
	}
}
