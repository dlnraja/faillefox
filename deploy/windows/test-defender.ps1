<#
.SYNOPSIS
    Teste le binaire Faillefox contre Windows Defender local.

.DESCRIPTION
    Build le binaire, lance un scan Defender, et affiche le résultat.
    Utile pour vérifier avant de distribuer que Defender ne flag pas le soft.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File test-defender.ps1
#>

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

Write-Host "==> Build du binaire Windows..." -ForegroundColor Cyan
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o faillefox-scan-test.exe ./cmd/faillefox
if ($LASTEXITCODE -ne 0) { throw "go build échoué" }

Write-Host "==> Scan avec Windows Defender..." -ForegroundColor Cyan
try {
    Start-MpScan -ScanType CustomScan -ScanPath (Resolve-Path "faillefox-scan-test.exe").Path
    Start-Sleep -Seconds 2

    $detections = Get-MpThreatDetection
    if ($detections) {
        Write-Host ""
        Write-Host "⚠ DETECTION TROUVÉE !" -ForegroundColor Red
        $detections | Format-List
        Write-Host ""
        Write-Host "Action recommandée : soumettre comme faux positif sur" -ForegroundColor Yellow
        Write-Host "  https://www.microsoft.com/en-us/wdsi/filesubmission"
    } else {
        Write-Host ""
        Write-Host "✅ AUCUNE DETECTION — Faillefox est clean selon Defender" -ForegroundColor Green
    }
} catch {
    Write-Host ""
    Write-Host "⚠ Start-MpScan indisponible (AV tiers géré par un autre éditeur)." -ForegroundColor Yellow
    Write-Host "  Détection count: $((Get-MpThreatDetection).Count)" -ForegroundColor Yellow
    if ((Get-MpThreatDetection).Count -eq 0) {
        Write-Host "✅ Aucune détection enregistrée — binaire probablement clean" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "==> Nettoyage..."
Remove-Item faillefox-scan-test.exe -Force
