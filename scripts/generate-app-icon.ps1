$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$buildIcon = Join-Path $root "build\appicon.png"
$frontendIcon = Join-Path $root "frontend\src\assets\images\appicon.png"
$windowsIcon = Join-Path $root "build\windows\icon.ico"

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $buildIcon) | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $frontendIcon) | Out-Null

$size = 1024
$bitmap = New-Object System.Drawing.Bitmap($size, $size)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit
$graphics.Clear([System.Drawing.Color]::Transparent)

$radius = 210
$diameter = $radius * 2
$path = New-Object System.Drawing.Drawing2D.GraphicsPath
$path.AddArc(0, 0, $diameter, $diameter, 180, 90)
$path.AddArc($size - $diameter, 0, $diameter, $diameter, 270, 90)
$path.AddArc($size - $diameter, $size - $diameter, $diameter, $diameter, 0, 90)
$path.AddArc(0, $size - $diameter, $diameter, $diameter, 90, 90)
$path.CloseFigure()

$background = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 17, 24, 39))
$graphics.FillPath($background, $path)

$accent = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 99, 102, 241))
$graphics.FillEllipse($accent, 704, 116, 184, 184)

$font = New-Object System.Drawing.Font("Segoe UI", 338, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$textBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
$text = "C+"
$textSize = $graphics.MeasureString($text, $font)
$x = ($size - $textSize.Width) / 2
$y = ($size - $textSize.Height) / 2 - 8
$graphics.DrawString($text, $font, $textBrush, $x, $y)

$bitmap.Save($buildIcon, [System.Drawing.Imaging.ImageFormat]::Png)
$bitmap.Save($frontendIcon, [System.Drawing.Imaging.ImageFormat]::Png)

$textBrush.Dispose()
$font.Dispose()
$accent.Dispose()
$background.Dispose()
$path.Dispose()
$graphics.Dispose()
$bitmap.Dispose()

# Wails only regenerates this file when it is missing. Remove any previously
# generated default Wails icon so the next build derives it from appicon.png.
if (Test-Path $windowsIcon) {
    Remove-Item -Force $windowsIcon
}

Write-Host "Generated CodexPro+ app icons:"
Write-Host "  $buildIcon"
Write-Host "  $frontendIcon"
