# Compilation & build

Guide complet pour compiler Faillefox sur chaque plateforme, en binaire
autonome ou via les CI GitHub Actions.

## Prérequis

- **Go 1.26+** : https://go.dev/dl/
- (Optionnel) **Git** pour cloner le dépôt
- (Optionnel, Android) **Android Studio** + NDK + `gomobile`

Vérifiez votre installation :

```bash
go version   # doit afficher go1.26 ou plus
```

## Build rapide (depuis les sources)

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox

# Binaire pour votre OS courant (dans le dossier courant)
go build -o faillefox ./cmd/faillefox

# Lancer
./faillefox            # Linux / macOS
faillefox.exe          # Windows
```

Puis ouvrez **http://127.0.0.1:8443**.

## Build optimisé (binaire plus petit)

Les binaires de release utilisent ces flags pour réduire la taille et
retirer les infos de debug :

```bash
go build -trimpath -ldflags="-s -w" -o faillefox ./cmd/faillefox
```

| Flag | Effet |
|------|-------|
| `-trimpath` | Supprime les chemins absolus de votre machine du binaire |
| `-ldflags="-s"` | Supprime la table des symboles |
| `-ldflags="-w"` | Supprime les infos de debug DWARF |

Gain typique : ~9 Mo → ~6,5 Mo.

## Cross-compilation

Go permet de compiler pour n'importe quelle cible depuis n'importe quel OS,
sans toolchain supplémentaire (CGO désactivé) :

```bash
# Windows amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faillefox-windows-amd64.exe ./cmd/faillefox

# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faillefox-linux-amd64 ./cmd/faillefox

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faillefox-darwin-arm64 ./cmd/faillefox
```

### Toutes les cibles supportées (release officielle)

| GOOS | GOARCH | Suffixe | Notes |
|------|--------|---------|-------|
| windows | amd64 | `.exe` | x86_64, le plus courant |
| windows | arm64 | `.exe` | Windows on ARM |
| linux | amd64 | — | x86_64 |
| linux | arm64 | — | Raspberry Pi 4/5, serveurs ARM |
| darwin | amd64 | — | Intel Mac |
| darwin | arm64 | — | Apple Silicon (M1/M2/M3) |

## Métadonnées Windows (VERSIONINFO)

Le binaire Windows embarque des métadonnées (Company, Product, Version,
manifeste UAC) qui réduisent les faux positifs antivirus. Elles sont
générées par `goversioninfo` depuis `version_info.json` :

```bash
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
cd cmd/faillefox
$HOME/go/bin/goversioninfo ../../version_info.json   # produit resource.syso
cd ../..
GOOS=windows GOARCH=amd64 go build -o faillefox.exe ./cmd/faillefox
```

Le fichier `resource.syso` est détecté et lié automatiquement par `go build`
(convention Go). Il est ignoré par git (régénéré en CI).

## Build Android

L'app Android encapsule le cœur Go via `gomobile`. Voir
[docs/android.md](android.md) pour le détail.

Résumé :

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
gomobile bind -target=android/amd64,android/arm64 \
    -o android/app/libs/faillefox.aar ./pkg/android

cd android && ./gradlew assembleDebug
# -> android/app/build/outputs/apk/debug/app-debug.apk
```

## Vérifications avant de pousser

```bash
go build ./...     # compile tous les packages (ne produit pas de binaire)
go vet ./...       # analyse statique
go test ./...      # tous les tests unitaires
```

Les 3 doivent passer sans erreur. C'est ce que vérifie la CI sur chaque PR.

## Build via CI (GitHub Actions)

Vous n'avez normalement **pas** à builder à la main : la CI le fait.

- **Sur chaque PR/push master** : `.github/workflows/ci.yml` build + vet + test
  sur matrice Ubuntu / Windows / macOS.
- **Sur chaque tag `v*`** : `.github/workflows/release.yml` cross-compile les
  6 binaires, génère le VERSIONINFO Windows, et publie la release.
- **À chaque merge master** : `.github/workflows/auto-version.yml` bump la
  version + CHANGELOG automatiquement (voir [docs/release.md](release.md)).

## Dépendances externes

Faillefox reste délibérément léger. Une seule dépendance non-stdlib :

| Dépendance | Rôle | Licence |
|------------|------|---------|
| `github.com/miekg/dns` | Moteur DNS (résolveur sinkhole) | BSD-3-Clause |

Mises à jour automatiques par **Dependabot** (PR hebdomadaires).

## Problèmes fréquents

### `go: module requires Go 1.26`
Votre Go est trop ancien. Mettez à jour depuis https://go.dev/dl/.

### `-race requires cgo` (sur Windows sans gcc)
Le détecteur de data races nécessite un compilateur C. Soit installez
`gcc` (MinGW), soit lancez `go test` sans `-race` (le CI le fait sur Linux).

### Le binaire est signalé par mon antivirus
Voir [docs/antivirus.md](antivirus.md) — Faillefox écoute un port local et
pilote le pare-feu système, ce qui peut déclencher des faux positifs. La
signature SignPath + la soumission aux labs règlent le problème.
