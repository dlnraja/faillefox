<#
.SYNOPSIS
    Ouvre TOUS les formulaires de soumission antivirus dans le navigateur par
    défaut, pour soumettre Faillefox comme faux positif.

.DESCRIPTION
    Ce script ouvre une série d'onglets dans votre navigateur avec les
    formulaires de soumission des principaux éditeurs antivirus. Il copie
    aussi dans le presse-papier un commentaire type à coller dans chaque
    formulaire.

    Vous n'avez plus qu'à :
      1. Uploader le fichier faillefox-windows-amd64.exe dans chaque formulaire
      2. Coller le commentaire (Ctrl+V) déjà dans le presse-papier
      3. Valider

.NOTES
    Lancement : powershell -ExecutionPolicy Bypass -File submit-to-av.ps1
#>

$comment = @"
Open-source firewall (GPL-3.0).
Source: https://github.com/dlnraja/faillefox
No telemetry, no network exfiltration.

The tool listens on 127.0.0.1 (loopback only) for its control panel and
pilots the Windows Firewall via netsh advfirewall. These are legitimate
security tool actions. This is a false positive.

SHA256: (voir le fichier SHA256SUMS dans la release)
Version: (voir la release GitHub)
"@

# Copie le commentaire type dans le presse-papier.
Set-Clipboard -Value $comment
Write-Host "Commentaire type copié dans le presse-papier (Ctrl+V dans chaque formulaire)." -ForegroundColor Green
Write-Host ""

# Liste des formulaires AV (par ordre d'impact).
$forms = @(
    @{ Name = "Microsoft Defender";      Url = "https://www.microsoft.com/en-us/wdsi/filesubmission" },
    @{ Name = "VirusTotal (70+ AV)";     Url = "https://www.virustotal.com/gui/home/upload" },
    @{ Name = "Kaspersky OpenTIP";       Url = "https://opentip.kaspersky.com/" },
    @{ Name = "Bitdefender";             Url = "https://www.bitdefender.com/analyze-sample.html" },
    @{ Name = "ESET";                    Url = "https://www.eset.com/int/home/support/false-positive/" },
    @{ Name = "Avast / AVG";             Url = "https://www.avast.com/false-positive-file-form.php" },
    @{ Name = "Sophos";                  Url = "https://www.sophos.com/en-us/legal/sophos-analysis-results" },
    @{ Name = "Trend Micro";             Url = "https://www.trendmicro.com/en_us/business/products/validation/filing.html" },
    @{ Name = "Malwarebytes";            Url = "https://forums.malwarebytes.com/forum/127-false-positives/" },
    @{ Name = "Comodo";                  Url = "https://submit.alliance.virusradar.com/" },
    @{ Name = "F-Secure";                Url = "https://www.f-secure.com/en/business/support/tools/labs-tool" },
    @{ Name = "Avira";                   Url = "https://www.avira.com/en/analysis" },
    @{ Name = "G Data";                  Url = "https://www.gdatasoftware.com/submit-malware" }
)

Write-Host "Ouverture de $($forms.Count) formulaires AV dans le navigateur..." -ForegroundColor Cyan
Write-Host ""
Write-Host "Priorité 1 (impact maximal) :"
Write-Host "  - Microsoft Defender (AV par défaut sur Windows)"
Write-Host "  - VirusTotal (70+ AV synchronisés d'un coup)"
Write-Host "  - Kaspersky OpenTIP"
Write-Host ""

foreach ($form in $forms) {
    Write-Host "  Ouverture : $($form.Name)" -ForegroundColor Yellow
    Start-Process $form.Url
    Start-Sleep -Milliseconds 800  # évite de saturer le navigateur
}

Write-Host ""
Write-Host "Termine ! $([forms.Count]) formulaires ouverts." -ForegroundColor Green
Write-Host "Prochaine etape dans chaque onglet :" -ForegroundColor Cyan
Write-Host "  1. Upload du fichier faillefox-windows-amd64.exe"
Write-Host "  2. Coller le commentaire (Ctrl+V)"
Write-Host "  3. Valider"
Write-Host ""
Write-Host "Conseil : commencez par Microsoft Defender et VirusTotal (impact maximal)."
Write-Host "Resultats attendus sous 2-14 jours selon l'editeur."
