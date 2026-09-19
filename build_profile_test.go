package main

import "testing"

func withBuildProfile(t *testing.T, profile string) {
	t.Helper()
	previous := buildProfile
	buildProfile = profile
	t.Cleanup(func() {
		buildProfile = previous
	})
}

func TestReleaseBuildProfileDefaults(t *testing.T) {
	withBuildProfile(t, "release")

	if got := currentBuildProfile(); got != releaseBuildProfile {
		t.Fatalf("currentBuildProfile() = %q, want %q", got, releaseBuildProfile)
	}
	if isDevBuild() {
		t.Fatal("release build reported dev profile")
	}
	if got := appDisplayName(); got != "CodexPro+" {
		t.Fatalf("appDisplayName() = %q, want %q", got, "CodexPro+")
	}
	if got := currentConfigDirectory(); got != configDirectory {
		t.Fatalf("currentConfigDirectory() = %q, want %q", got, configDirectory)
	}
	if got := managerRunValueName(); got != releaseManagerRunName {
		t.Fatalf("managerRunValueName() = %q, want %q", got, releaseManagerRunName)
	}
	if got := desktopAppID(); got != releaseDesktopAppID {
		t.Fatalf("desktopAppID() = %q, want %q", got, releaseDesktopAppID)
	}
}

func TestDevBuildProfileIsIsolated(t *testing.T) {
	withBuildProfile(t, "DEV")

	if got := currentBuildProfile(); got != devBuildProfile {
		t.Fatalf("currentBuildProfile() = %q, want %q", got, devBuildProfile)
	}
	if !isDevBuild() {
		t.Fatal("dev build did not report dev profile")
	}
	if got := appDisplayName(); got != "CodexPro+ Dev" {
		t.Fatalf("appDisplayName() = %q, want %q", got, "CodexPro+ Dev")
	}
	if got := currentConfigDirectory(); got != devConfigDirectory {
		t.Fatalf("currentConfigDirectory() = %q, want %q", got, devConfigDirectory)
	}
	if got := managerRunValueName(); got != devManagerRunName {
		t.Fatalf("managerRunValueName() = %q, want %q", got, devManagerRunName)
	}
	if got := desktopAppID(); got != devDesktopAppID {
		t.Fatalf("desktopAppID() = %q, want %q", got, devDesktopAppID)
	}
}

func TestUnknownBuildProfileFallsBackToRelease(t *testing.T) {
	withBuildProfile(t, "something-else")

	if got := currentBuildProfile(); got != releaseBuildProfile {
		t.Fatalf("currentBuildProfile() = %q, want release fallback", got)
	}
}
