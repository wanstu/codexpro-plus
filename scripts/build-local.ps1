param(
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$outputName = "codexpro-plus-dev.exe"
$outputPath = Join-Path $root "build\bin\$outputName"
$corePath = Join-Path $root "build\bin\codexpro-core.exe"

Push-Location $root
try {
    & (Join-Path $PSScriptRoot "generate-app-icon.ps1")

    if (-not $SkipTests) {
        & go test ./...
        if ($LASTEXITCODE -ne 0) {
            throw "go test ./... failed with exit code $LASTEXITCODE"
        }
    }

    & wails build -o $outputName -ldflags "-X main.buildProfile=dev"
    if ($LASTEXITCODE -ne 0) {
        throw "wails build failed with exit code $LASTEXITCODE"
    }

    if (-not (Test-Path $outputPath)) {
        throw "Local Dev build output was not created: $outputPath"
    }

    Write-Host ""
    Write-Host "CodexPro+ Dev build ready:"
    Write-Host "  exe:    $outputPath"
    Write-Host "  config: ~/.config/codexpro-plus-dev/config.json"
    Write-Host "  title:  CodexPro+ Dev"
    Write-Host ""
    Write-Host "This script only builds the Dev executable; it does not launch it."

    if (-not (Test-Path $corePath)) {
        Write-Warning "codexpro-core.exe is not present next to the Dev executable. Build/copy it before starting a Workspace."
    }
}
finally {
    Pop-Location
}
