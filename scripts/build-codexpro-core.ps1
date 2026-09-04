param(
    [string]$Source,
    [string]$Output = (Join-Path $PSScriptRoot "..\build\bin\codexpro-core.exe"),
    [string]$Ref = "587f7fd3a4644a847bba13aeb49336056052e1f6"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoUrl = "https://github.com/rebel0789/codexpro.git"
$outputPath = [System.IO.Path]::GetFullPath($Output)
$tempSource = $null

function Require-Command([string]$Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

Require-Command git
Require-Command npm
Require-Command bun

try {
    if ([string]::IsNullOrWhiteSpace($Source)) {
        $tempSource = Join-Path ([System.IO.Path]::GetTempPath()) ("codexpro-plus-core-" + [guid]::NewGuid().ToString("N"))
        Write-Host "Cloning CodexPro $Ref ..."
        git clone --filter=blob:none --no-checkout $repoUrl $tempSource
        if ($LASTEXITCODE -ne 0) { throw "git clone failed" }

        git -C $tempSource fetch --depth 1 origin $Ref
        if ($LASTEXITCODE -ne 0) { throw "git fetch $Ref failed" }

        git -C $tempSource checkout --detach FETCH_HEAD
        if ($LASTEXITCODE -ne 0) { throw "git checkout $Ref failed" }

        $Source = $tempSource
    } else {
        $Source = [System.IO.Path]::GetFullPath($Source)
        if (-not (Test-Path (Join-Path $Source "package.json"))) {
            throw "CodexPro source directory does not contain package.json: $Source"
        }
        Write-Host "Using local CodexPro source: $Source"
    }

    New-Item -ItemType Directory -Force -Path (Split-Path $outputPath -Parent) | Out-Null

    Push-Location $Source
    try {
        Write-Host "Installing CodexPro dependencies ..."
        npm ci --ignore-scripts --no-audit --no-fund
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }

        Write-Host "Building CodexPro TypeScript ..."
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }

        Write-Host "Compiling standalone codexpro-core.exe with Bun ..."
        bun build --compile .\dist\http.js --outfile $outputPath
        if ($LASTEXITCODE -ne 0) { throw "bun build --compile failed" }
    } finally {
        Pop-Location
    }

    if (-not (Test-Path $outputPath)) {
        throw "codexpro-core.exe was not created: $outputPath"
    }

    $size = (Get-Item $outputPath).Length
    Write-Host "Built: $outputPath ($size bytes)"
} finally {
    if ($tempSource -and (Test-Path $tempSource)) {
        Remove-Item $tempSource -Recurse -Force -ErrorAction SilentlyContinue
    }
}
