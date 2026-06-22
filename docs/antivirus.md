# Guide antivirus & signature — réduire les faux positifs

> ⚠️ **Honnêteté fondamentale** : il est **impossible de garantir 0 alerte
> antivirus**, même pour Microsoft ou Google. Ce guide décrit les actions
> CONCRÈTES qui réduisent les faux positifs de façon mesurable.

---

## Ce qui est déjà en place dans Faillefox

| Mesure | Statut |
|--------|--------|
| VERSIONINFO complète (Company, Product, Version) | ✅ via `goversioninfo` |
| Manifeste UAC `asInvoker` | ✅ |
| Signature self-signed (best-effort) | ✅ dans le workflow release |
| Auto-soumission VirusTotal | ✅ workflow `av-reputation.yml` |
| Bind explicite `127.0.0.1` (pas `0.0.0.0`) | ✅ |
| Sanitization des entrées (anti-injection) | ✅ |
| Workflow SignPath prêt à brancher | ✅ |

---

## 1. SignPath — signature Authenticode GRATUITE (la voie la plus efficace)

[SignPath Foundation](https://signpath.org/) offre une signature
**Authenticode reconnue** (pas self-signed) **gratuitement** aux projets
open source vérifiables. C'est la seule voie gratuite qui a un vrai impact.

### Étape 1 : Créer un compte organisation

1. Allez sur https://signpath.org/
2. Cliquez **« Request free code signing »** (en bas de page)
3. Remplissez le formulaire :
   - Organization name : `dlnraja` (ou votre nom GitHub)
   - Project name : `Faillefox`
   - Repository URL : `https://github.com/dlnraja/faillefox`
   - License : `GPL-3.0`
4. SignPath vérifie que le dépôt vous appartient (vous devez être owner).

### Étape 2 : Configurer le projet dans SignPath

Une fois approuvé (24-72h) :

1. Connectez-vous sur https://app.signpath.org/
2. **Projects** → **Add project** :
   - Name : `Faillefox`
   - Integration : **GitHub**
   - Repository : `dlnraja/faillefox`
3. **Signing policies** → **Add policy** :
   - Name : `release`
   - Certificate : sélectionnez le certificat OV proposé par SignPath
   - Artifact configuration : `faillefox-windows-amd64.exe`
4. Récupérez votre **Organization ID** (Profile → Organization).

### Étape 3 : Configurer les secrets GitHub

Dans votre dépôt GitHub → **Settings** → **Secrets and variables** → **Actions** → **New repository secret** :

| Secret name | Valeur |
|-------------|--------|
| `SIGNPATH_API_TOKEN` | Votre API token (Profile → API tokens → Generate) |
| `SIGNPATH_ORGANIZATION_ID` | Votre Organization ID |
| `SIGNPATH_PROJECT_SLUG` | `faillefox` |
| `SIGNPATH_POLICY_SLUG` | `release` |

Puis dans **Variables** (pas secrets) :
| Variable name | Valeur |
|---------------|--------|
| `SIGNPATH_ENABLED` | `true` |

### Étape 4 : Le workflow signe automatiquement

Le workflow `.github/workflows/sign.yml` existe déjà. Dès que
`SIGNPATH_ENABLED=true` + les secrets sont configurés, **chaque release**
signera automatiquement le `.exe` Windows avec un certificat reconnu.

### Résultat

Après signature SignPath :
- Windows SmartScreen affiche **« Verified publisher: dlnraja »** au lieu de
  **« Unknown publisher »**
- La réputation du binaire monte plus vite (moins de faux positifs)
- Microsoft Defender fait davantage confiance au binaire signé

---

## 2. Auto-soumission VirusTotal (déjà configurée)

Le workflow `av-reputation.yml` soumet automatiquement chaque release à
VirusTotal. **Pourquoi ça aide** : les 70+ AV se synchronisent sur VT. Plus
un binaire y est analysé sans détection, plus sa réputation monte.

### Activation
1. Créer un compte gratuit sur https://www.virustotal.com
2. Profile → **API Key** → copier
3. GitHub → Settings → Secrets → `VIRUSTOTAL_API_KEY` = votre clé

---

## 3. Soumission manuelle Microsoft Defender

Microsoft Defender est l'AV le plus important sur Windows. La soumission
**manuelle** est la voie la plus efficace pour les faux positifs Defender :

1. Allez sur https://www.microsoft.com/en-us/wdsi/filesubmission
2. Connectez-vous avec un compte Microsoft
3. Upload du `faillefox-windows-amd64.exe`
4. Commentaire :
   > *Open-source firewall (GPL-3.0). Source: github.com/dlnraja/faillefox.
   > No telemetry, no network exfiltration. The tool listens on 127.0.0.1
   > for its control panel and pilots the Windows Firewall via netsh. This
   > is a false positive. Thank you.*
5. Microsoft analyse sous **2-14 jours** et désflague le binaire.

---

## 4. Autres labs AV (soumission manuelle)

| Éditeur | URL |
|---------|-----|
| Kaspersky | https://opentip.kaspersky.com/ |
| ESET | https://www.eset.com/int/home/support/false-positive/ |
| Bitdefender | https://www.bitdefender.com/analyze-sample.html |
| Avast/AVG | https://www.avast.com/false-positive-file-form.php |
| Sophos | https://www.sophos.com/en-us/legal/sophos-analysis-results |
| Malwarebytes | https://forums.malwarebytes.com/forum/127-false-positives/ |

---

## 5. Limites honnêtes

| Approche | Efficacité | Coût |
|----------|------------|------|
| SignPath (signature reconnue) | ⭐⭐⭐⭐ | Gratuit (OSS) |
| Soumission VirusTotal | ⭐⭐⭐ | Gratuit |
| Soumission Defender manuelle | ⭐⭐⭐⭐ | Gratuit |
| Self-signed cert | ⭐ | Gratuit (best-effort) |
| Certificat OV payant | ⭐⭐⭐⭐⭐ | ~300-600 €/an |
| Certificat EV payant | ⭐⭐⭐⭐⭐ | ~600-900 €/an |

**Aucune de ces approches ne garantit 0 alerte.** Même les binaires signés EV
peuvent être signalés par un AV heuristique. La combinaison
**SignPath + VirusTotal + soumission Defender** donne les meilleurs résultats
gratuits possibles.

---

## 6. Vérifier une signature

Après signature, vous pouvez vérifier sous Windows :
```cmd
powershell Get-AuthenticodeSignature faillefox.exe
```
Doit afficher `Status: Valid` et le signataire.

Sous Linux pour analyser le `.exe` :
```bash
osslsigncode verify -in faillefox.exe
```
