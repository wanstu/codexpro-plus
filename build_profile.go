package main

import "strings"

// Local source builds default to Dev so they can safely run beside the installed/release build.
// GitHub/Release builds explicitly override this to release via -ldflags.
var buildProfile = "dev"

const (
	releaseBuildProfile   = "release"
	devBuildProfile       = "dev"
	configDirectory       = "codexpro-plus"
	devConfigDirectory    = "codexpro-plus-dev"
	releaseManagerRunName = "CodexProPlus"
	devManagerRunName     = "CodexProPlusDev"
)

func currentBuildProfile() string {
	if strings.EqualFold(strings.TrimSpace(buildProfile), devBuildProfile) {
		return devBuildProfile
	}
	return releaseBuildProfile
}

func isDevBuild() bool {
	return currentBuildProfile() == devBuildProfile
}

func appDisplayName() string {
	if isDevBuild() {
		return "CodexPro+ Dev"
	}
	return "CodexPro+"
}

func currentConfigDirectory() string {
	if isDevBuild() {
		return devConfigDirectory
	}
	return configDirectory
}

func managerRunValueName() string {
	if isDevBuild() {
		return devManagerRunName
	}
	return releaseManagerRunName
}
