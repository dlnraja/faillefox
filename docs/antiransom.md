# Protection anti-ransomware

Faillefox intègre une couche de **détection comportementale** des
rançongiciels (ransomware). Ce n'est pas un moteur heuristique ML — c'est
une détection par règles simples mais efficaces, qui complète une vraie
solution EDR.

## Activation

```bash
./faillefox -antiransom
```

Ou via le panneau de paramètres (onglet ⚙️ Paramètres → mode avancé →
Anti-ransomware).

## Mécanismes de détection

Le module `internal/antiransom` surveille les dossiers sensibles et déclenche
des alertes sur 3 types de signaux :

### 1. Ransom notes (CRITICAL)
Détecte la création de fichiers aux noms caractéristiques laissés par les
ransomwares pour exiger la rançon :
- `README_DECRYPT.txt`, `HOW_TO_DECRYPT.html`, `!RESTORE_FILES.txt`
- `lockbit_readme.txt`, `conti_readme`, `ryuk_readme`...

~15 patterns de familles actives (LockBit, Conti, Ryuk, BlackCat, Royal...).

### 2. Extensions chiffrées (WARNING)
Détecte les fichiers portant une extension associée au chiffrement par
ransomware :
- `.lockbit`, `.conti`, `.ryuk`, `.blackcat`, `.royal`, `.akira`
- `.locked`, `.crypto`, `.encrypted`, `.enc`, `.phobos`...

~25 extensions connues.

### 3. Rate-limiting (CRITICAL)
Détecte une activité d'écriture anormale dans un dossier sensible —
comportement typique d'un chiffrement en masse :
- **Seuil** : > 200 fichiers modifiés en 30 secondes
- **Dossiers surveillés** : Documents, Pictures, Desktop, Downloads (+ OneDrive sur Windows)
- Déclenche une alerte CRITICAL, puis reset le compteur (anti-rafale)

## Surveillance active (fsnotify)

Depuis la v0.12, le module utilise `fsnotify` pour surveiller les dossiers
sensibles **en temps réel**. Chaque création/écriture/renommage est transmis
au détecteur.

Limitation : `fsnotify` peut générer beaucoup d'événements sur un dossier très
actif (compilations, sync cloud). Le rate-limiting du détecteur absorbe le
bruit et n'alerte que sur les volumes anormaux.

## Niveaux d'alerte

| Niveau | Déclencheur | Action recommandée |
|--------|-------------|--------------------|
| **WARNING** | Extension chiffrée détectée | Vérifier le fichier, scanner avec ClamAV |
| **CRITICAL** | Ransom note OU chiffrement massif | **Déconnecter le réseau immédiatement**, isoler la machine, restaurer depuis backup |

## ⚠️ Limitations honnêtes

- **Pas de moteur heuristique ML** : ne peut pas identifier un ransomware
  zero-day inconnu par son comportement algorithmique. Ce serait impossible
  seul (nécessite infrastructure de ML + corpus de malwares).
- **Ne restaure pas les fichiers** : la restauration nécessite des backups
  ou shadow copies (hors scope d'un pare-feu). **Faites des backups réguliers.**
- **fsnotify peut manquer des événements** si le volume est très élevé.
  Rationnel : on alerte dès le seuil atteint, même sur un sous-ensemble.
- **Ne remplace pas une solution EDR commerciale** (CrowdStrike, SentinelOne,
  Microsoft Defender for Endpoint). C'est une couche de détection additionnelle.

## Architecture

```
fsnotify watcher → OnFileEvent(path)
                     ├─ checkRansomNote(path)    → CRITICAL si match
                     ├─ checkEncryptedExt(path)  → WARNING si match
                     └─ rate-limiting par dossier → CRITICAL si seuil dépassé
```

Code source : `internal/antiransom/`
