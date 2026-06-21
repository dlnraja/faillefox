# build-installer.ps1 — Build le binaire Windows puis l'installateur .exe.
#
# Prérequis :
#   - Go 1.26+
#   - Inno Setup (iscc) dans le PATH  ->  https://jrsoftware.org/isdl.php
#
# Usage (depuis la racine du dépôt) :
#   powershell -File deploy/windows/build-installer.ps1
#
# Produit : deploy/windows/Output/faillefox-setup-<version>.exe
param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

# 1. Version : depuis le paramètre, sinon depuis le dernier tag git.
if (-not $Version) {
    $gitTag = (git describe --tags --abbrev=0 2>$null)
    if ($gitTag) { $Version = $gitTag.TrimStart("v") } else { $Version = "0.0.0" }
}
Write-Host "==> Version: $Version"

# 2. Build du binaire Windows amd64.
Write-Host "==> Compilation faillefox.exe (windows/amd64)"
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o faillefox.exe ./cmd/faillefox
if ($LASTEXITCODE -ne 0) { throw "go build échoué" }

# 3. VERSIONINFO Windows (goversioninfo -> resource.syso lié au prochain build).
$gvi = "$env:USERPROFILE\go\bin\goversioninfo.exe"
if (Test-Path $gvi) {
    Write-Host "==> Génération VERSIONINFO (goversioninfo)"
    Push-Location cmd/faillefox
    & $gvi -skip-versioninfo `
        "-ver-file-major=$($Version.Split('.')[0])" `
        "-ver-file-minor=$($Version.Split('.')[1])" `
        "-ver-file-patch=$($Version.Split('.')[2].Split('-')[0])" `
        -company-name="dlnraja" -product-name="Faillefox" `
        -file-description="Faillefox - Pare-feu multiplateforme" `
        -copyright="(c) 2026 dlnraja - GPL-3.0" `
        -original-filename="faillefox.exe" ../../version_info.json
    Pop-Location
    # Rebuild pour embarquer le .syso.
    go build -trimpath -ldflags="-s -w" -o faillefox.exe ./cmd/faillefox
}

# 4. Compilation de l'installateur via Inno Setup.
$iscc = Get-Command iscc -ErrorAction SilentlyContinue
if (-not $iscc) {
    $isccPath = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
    if (Test-Path $isccPath) { $iscc = @{ Path = $isccPath } }
}
if (-not $iscc) {
    Write-Warning "Inno Setup (iscc) introuvable. Binaire seul produit : faillefox.exe"
    Write-Warning "Installez Inno Setup depuis https://jrsoftware.org/isdl.php puis relancez."
    exit 0
}
Write-Host "==> Compilation installateur (Inno Setup)"
& $iscc.Path "/DMyAppVersion=$Version" deploy/windows/faillefox.iss
if ($LASTEXITCODE -ne 0) { throw "iscc échoué" }

$out = "deploy/windows/Output/faillefox-setup-$Version.exe"
if (Test-Path $out) {
    $size = [math]::Round((Get-Item $out).Length / 1MB, 1)
    Write-Host "==> OK: $out ($size Mo)"
} else {
    throw "Installateur non produit"
}
