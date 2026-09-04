$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

$appSource = Join-Path $root "assets\icons\codexpro-plus-app.png"
$traySource = Join-Path $root "assets\icons\codexpro-plus-tray.png"
$windowSource = Join-Path $root "assets\icons\codexpro-plus-window.png"

$buildIcon = Join-Path $root "build\appicon.png"
$frontendAppIcon = Join-Path $root "frontend\src\assets\images\appicon.png"
$frontendTrayIcon = Join-Path $root "frontend\src\assets\images\tray-icon.png"
$windowsIcon = Join-Path $root "build\windows\icon.ico"

foreach ($source in @($appSource, $traySource, $windowSource)) {
    if (-not (Test-Path $source)) {
        throw "Missing icon source: $source"
    }
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $buildIcon) | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $frontendAppIcon) | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $frontendTrayIcon) | Out-Null

# Wails derives the native Windows icon from build/appicon.png.
# Use the dedicated window/taskbar artwork for that native resource.
Copy-Item -Force $windowSource $buildIcon

# Keep the app artwork available to the embedded frontend and use the
# dedicated tray artwork for the Windows system tray.
Copy-Item -Force $appSource $frontendAppIcon
Copy-Item -Force $traySource $frontendTrayIcon

# Wails only regenerates build/windows/icon.ico when it is missing.
# Remove the previous resource so the next build derives it from the new
# window/taskbar source above.
if (Test-Path $windowsIcon) {
    Remove-Item -Force $windowsIcon
}

Write-Host "Prepared CodexPro+ icon assets:"
Write-Host "  app:    $frontendAppIcon"
Write-Host "  tray:   $frontendTrayIcon"
Write-Host "  window: $buildIcon"
