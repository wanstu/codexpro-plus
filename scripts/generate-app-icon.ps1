$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing

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

function Export-FittedSquareIcon {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination,
        [double]$Fill = 0.98
    )

    $sourceBitmap = [System.Drawing.Bitmap]::FromFile($Source)
    try {
        $canvasSize = 1024
        $targetSize = $canvasSize * $Fill
        $scale = [Math]::Min($targetSize / $sourceBitmap.Width, $targetSize / $sourceBitmap.Height)
        $drawWidth = [int][Math]::Round($sourceBitmap.Width * $scale)
        $drawHeight = [int][Math]::Round($sourceBitmap.Height * $scale)
        $drawX = [int][Math]::Round(($canvasSize - $drawWidth) / 2)
        $drawY = [int][Math]::Round(($canvasSize - $drawHeight) / 2)

        $output = New-Object System.Drawing.Bitmap($canvasSize, $canvasSize, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
        try {
            $graphics = [System.Drawing.Graphics]::FromImage($output)
            try {
                $graphics.Clear([System.Drawing.Color]::Transparent)
                $graphics.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
                $graphics.DrawImage($sourceBitmap, $drawX, $drawY, $drawWidth, $drawHeight)
            }
            finally {
                $graphics.Dispose()
            }
            $output.Save($Destination, [System.Drawing.Imaging.ImageFormat]::Png)
        }
        finally {
            $output.Dispose()
        }
    }
    finally {
        $sourceBitmap.Dispose()
    }
}

# The current source artwork is already tightly cropped. Keep the entire mark,
# normalize it onto a square icon canvas, and let it occupy almost the full slot.
Export-FittedSquareIcon -Source $appSource -Destination $frontendAppIcon -Fill 0.98
Export-FittedSquareIcon -Source $windowSource -Destination $buildIcon -Fill 0.98

# Windows tray slots are square. A 603x325 horizontal logo will always look
# undersized vertically when aspect ratio is preserved. Like img-lock-v2, use a
# compact square mark for the tray instead of shrinking the wide brand lockup.
Export-FittedSquareIcon -Source $windowSource -Destination $frontendTrayIcon -Fill 0.99

# Wails regenerates the Windows ICO from build/appicon.png when this is absent.
if (Test-Path $windowsIcon) {
    Remove-Item -Force $windowsIcon
}

Write-Host "Prepared full-size CodexPro+ icon assets:"
Write-Host "  app:    $frontendAppIcon"
Write-Host "  tray:   $frontendTrayIcon"
Write-Host "  window: $buildIcon"
