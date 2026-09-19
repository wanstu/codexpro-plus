$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

$appSource = Join-Path $root "assets\icons\codexpro-plus-app.png"
$windowSource = Join-Path $root "assets\icons\codexpro-plus-window.png"
$frontendAppIcon = Join-Path $root "frontend\src\assets\images\appicon.png"
$frontendTrayIcon = Join-Path $root "frontend\src\assets\images\tray-icon.png"

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

    $args = @(
        "run", "github.com/wanstu/wails-desktop-kit/cmd/desktopkit",
        "icon", "normalize",
        "--input", $Source,
        "--output", $Destination,
        "--canvas", "1024",
        "--fill", $Fill.ToString([Globalization.CultureInfo]::InvariantCulture),
        "--trim-alpha=false"
    )
    & go @args
    if ($LASTEXITCODE -ne 0) {
        throw "Desktop Kit icon normalize failed for $Source"
    }
}

Push-Location $root
try {
    Invoke-DesktopKitIconNormalize -Source $appSource -Destination $frontendAppIcon -Fill 0.98
    Invoke-DesktopKitIconNormalize -Source $windowSource -Destination $frontendTrayIcon -Fill 0.99

    $prepareArgs = @(
        "run", "github.com/wanstu/wails-desktop-kit/cmd/desktopkit",
        "icon", "prepare-wails",
        "--input", $windowSource,
        "--desktop-dir", $root,
        "--normalize",
        "--canvas", "1024",
        "--fill", "0.98",
        "--trim-alpha=false"
    )
    & go @prepareArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Desktop Kit Wails icon preparation failed for $windowSource"
    }
}
finally {
    Pop-Location
}

Write-Host "Prepared CodexPro+ icons with Wails Desktop Kit:"
Write-Host "  app:    $frontendAppIcon"
Write-Host "  tray:   $frontendTrayIcon"
Write-Host ("  window: " + (Join-Path $root "build\appicon.png"))
