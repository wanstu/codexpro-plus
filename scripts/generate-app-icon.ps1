$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

$appSource = Join-Path $root "assets\icons\codexpro-plus-app.png"
$windowSource = Join-Path $root "assets\icons\codexpro-plus-window.png"

$buildIcon = Join-Path $root "build\appicon.png"
$frontendAppIcon = Join-Path $root "frontend\src\assets\images\appicon.png"
$frontendTrayIcon = Join-Path $root "frontend\src\assets\images\tray-icon.png"
$windowsIcon = Join-Path $root "build\windows\icon.ico"

foreach ($source in @($appSource, $windowSource)) {
    if (-not (Test-Path $source)) {
        throw "Missing icon source: $source"
    }
}

function Invoke-DesktopKitIconNormalize {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination,
        [Parameter(Mandatory = $true)][double]$Fill
    )

    & go run github.com/wanstu/wails-desktop-kit/cmd/desktopkit icon normalize `
        --input $Source `
        --output $Destination `
        --canvas 1024 `
        --fill $Fill `
        --trim-alpha=false

    if ($LASTEXITCODE -ne 0) {
        throw "Desktop Kit icon normalize failed for $Source"
    }
}

Push-Location $root
try {
    Invoke-DesktopKitIconNormalize -Source $appSource -Destination $frontendAppIcon -Fill 0.98
    Invoke-DesktopKitIconNormalize -Source $windowSource -Destination $buildIcon -Fill 0.98
    Invoke-DesktopKitIconNormalize -Source $windowSource -Destination $frontendTrayIcon -Fill 0.99
}
finally {
    Pop-Location
}

# Wails regenerates the Windows ICO from build/appicon.png when this is absent.
if (Test-Path $windowsIcon) {
    Remove-Item -Force $windowsIcon
}

Write-Host "Prepared CodexPro+ icons with Wails Desktop Kit:"
Write-Host "  app:    $frontendAppIcon"
Write-Host "  tray:   $frontendTrayIcon"
Write-Host "  window: $buildIcon"
