# Guide antivirus & signature

Faillefox est un logiciel de sécurité : il fait des choses qu'un antivirus
peut trouver suspectes par défaut (écouter un port local, interagir avec le
pare-feu système). Ce guide explique comment **faire reconnaître Faillefox**
par les antivirus afin de supprimer les faux positifs, de deux façons :

1. **Signature Authenticode** du binaire Windows (voie permanente)
2. **Soumission manuelle aux labs antivirus** (voie ponctuelle)

---

## 1. Pourquoi un faux positif ?

Les heuristiques antivirus se méfient des programmes qui :
- écoutent un port réseau (Faillefox écoute `127.0.0.1:8443` pour son UI),
- manipulent le pare-feu système (`netsh advfirewall` sur Windows),
- sont récents et peu téléchargés (faible « réputation »),
- ne sont **pas signés** (pas d'identité vérifiée).

Résultat : sans signature ni réputation, un antivirus comme Windows Defender
peut afficher *« Application non reconnue »* voire bloquer l'exécution.

## 2. La signature Authenticode (recommandé, gratuit via SignPath)

La signature Authenticode attache une identité vérifiée au `.exe` Windows.
Les antivirus font davantage confiance aux binaires signés.

### SignPath — signature gratuite pour l'open source

[SignPath Foundation](https://signpath.org/) offre la signature
Authenticode **gratuite** aux projets open source vérifiables. C'est la voie
choisie par Faillefox.

**Procédure (une fois)** :
1. Créer un compte sur https://signpath.org/ (compte organisation).
2. Demander l'accès gratuit OSS — SignPath vérifie que le dépôt GitHub
   `dlnraja/faillefox` vous appartient.
3. Dans SignPath : créer un **projet** Faillefox, puis une **policy**
   de signature.
4. Générer un **API token**.
5. Dans GitHub, onglet Settings → Secrets and variables → Actions, ajouter :
   - `SIGNPATH_API_TOKEN`
   - `SIGNPATH_ORGANIZATION_ID`
   - `SIGNPATH_PROJECT_SLUG` (= `faillefox`)
   - `SIGNPATH_POLICY_SLUG`
   - Variable `SIGNPATH_ENABLED` = `true`
6. Le workflow `.github/workflows/sign.yml` signera automatiquement chaque
   release. Le `.exe` signé sera remis dans la release.

### Si vous préférez un certificat payant

Un certificat code-signing OV/EV (~300-600 €/an, Sectigo / DigiCert) permet
de signer hors SignPath. Le workflow peut être adapté pour utiliser
`signtool` avec un secret `.pfx`.

---

## 3. Soumission aux labs antivirus (manuel, gratuit)

Même sans signature, vous pouvez soumettre le binaire aux principaux éditeurs
pour qu'ils l'analysent et le désflagent. Voici les portails :

| Éditeur | Portail de soumission | Portée |
|---------|----------------------|--------|
| **Microsoft Defender** | https://www.microsoft.com/en-us/wdsi/filesubmission | Windows Defender (le plus important sur Windows) |
| **Kaspersky** | https://opentip.kaspersky.com/ | Kaspersky + bases partagées |
| **VirusTotal** | https://www.virustotal.com/ | **70+ antivirus d'un coup** (Google, Symantec, ESET, Bitdefender…) |
| **ESET** | https://www.eset.com/int/home/support/false-positive/ | ESET NOD32, Smart Security |
| **Bitdefender** | https://www.bitdefender.com/analyze-sample.html | Bitdefender |
| **Avast / AVG** | https://www.avast.com/false-positive-file-form.php | Avast, AVG |
| **Sophos** | https://www.sophos.com/en-us/legal/sophos-analysis-results | Sophos |
| **Trend Micro** | https://www.trendmicro.com/en_us/business/products/validation/filing.html | Trend Micro |
| **Malwarebytes** | https://forums.malwarebytes.com/forum/127-false-positives/ | Malwarebytes |
| **Comodo** | https://www.comodo.com/home/internet-security/submit.php | Comodo |

### Procédure type pour chaque lab

1. Téléchargez l'archive de la release (le `.zip` Windows).
2. Sur le portail de l'éditeur, soumettez le `.exe` extrait.
3. Dans le commentaire, expliquez : **« Open-source firewall, GPL-3.0,
   source: https://github.com/dlnraja/faillefox, no telemetry, no network
   exfiltration. False positive. »**
4. La plupart désflagent sous **2 à 14 jours**.

### VirusTotal — la priorité

Soumettre à **VirusTotal** est le plus efficace : un seul upload et 70+
antivirus reçoivent l'échantillon + vos commentaires. Beaucoup d'éditeurs
se synchronisent sur VirusTotal.

> ⚠️ **Note** : soumettre un fichier à VirusTotal le rend public et
> indexable. C'est attendu pour un projet open source, mais soyez-en conscient.

---

## 4. Bonnes pratiques déjà appliquées dans Faillefox

Pour minimiser les faux positifs dès la conception :

- ✅ **Métadonnées VERSIONINFO** complètes (`version_info.json` → ressource
  `.syso`) : Company, Product, Version, Copyright.
- ✅ **Manifeste UAC** `asInvoker` (pas d'élévation cachée).
- ✅ **Source ouverte et auditable** (GPL-3.0).
- ✅ **Aucune télémétrie**, aucun appel réseau sortant du démon.
- ✅ **Loopback uniquement** pour le canal de contrôle (`127.0.0.1`).
- ✅ **Workflow SignPath** prêt à brancher (dès qu'on a les secrets).
- ✅ **Documentation honnête** : la v0.2 filtre réellement via
  `netsh advfirewall` / `nftables`, mais ne prétend pas être invulnérable.

---

## 5. Vérifier une signature

Après signature, vous pouvez vérifier sous Windows :
```cmd
powershell Get-AuthenticodeSignature faillefox.exe
```
Doit afficher `Status: Valid` et le signataire.

Sous Linux pour analyser le `.exe` :
```bash
osslsigncode verify -in faillefox.exe
```
