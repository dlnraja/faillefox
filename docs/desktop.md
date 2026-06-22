# Faillefox Desktop GUI

Faillefox dispose d'une **vraie application desktop native** (fenêtre avec
barre de titre, boutons fermer/réduire, redimensionnement) en plus du
panneau web et du TUI terminal.

## C'est quoi exactement ?

- Une fenêtre native de votre OS (Windows, Linux, macOS)
- Construite avec [Fyne](https://fyne.io) (framework GUI standard de Go)
- Widgets natifs : onglets, boutons, listes, barres de défilement
- Parle à l'API loopback du démon Faillefox (comme le panneau web)

## Installation

Téléchargez `faillefox-gui-<version>-<os>.zip` (ou `.tar.gz`) depuis la
[dernière release](https://github.com/dlnraja/faillefox/releases/latest).

## Utilisation

Le GUI est un **client** : il a besoin que le démon `faillefox` tourne en
arrière-plan.

```bash
# 1. Démarrer le démon (en service ou en console)
faillefox -dns -cve -threat-intel

# 2. Lancer le GUI
faillefox-gui
# ou avec port/token explicites :
faillefox-gui -port 8443 -token $(cat ~/.faillefox/token)
```

Le GUI récupère automatiquement le token depuis `~/.faillefox/token`.

## Onglets

| Onglet | Contenu |
|--------|---------|
| 🛡️ Dashboard | Statut du démon + score de protection |
| 📋 Règles | Liste des règles de filtrage |
| 🔐 Sécurité | Centre de sécurité (13 protections, statut de chacune) |
| 🧰 Outils | Scanner de ports + générateur de mot de passe |

## Compilation depuis les sources

Le GUI nécessite **CGO** + OpenGL (contrairement au démon qui est pur Go).

### Windows
```bash
# MinGW/gcc requis (déjà présent si vous avez installé Go via l'installeur officiel)
go install fyne.io/fyne/v2@latest
go build -o faillefox-gui.exe ./cmd/faillefox-gui
```

### Linux
```bash
sudo apt install libgl1-mesa-dev xorg-dev  # dépendances OpenGL/X11
go build -o faillefox-gui ./cmd/faillefox-gui
```

### macOS
```bash
# Xcode Command Line Tools requis (xcode-select --install)
go build -o faillefox-gui ./cmd/faillefox-gui
```

## Architecture

```
┌─────────────────────────────────────────┐
│  faillefox-gui (Fyne, CGO, fenêtre native) │
│  internal/desktop/app.go                │
└──────────────────┬──────────────────────┘
                   │ HTTP REST (loopback + token)
┌──────────────────▼──────────────────────┐
│  faillefox (démon, pur Go, pas CGO)     │
│  cmd/faillefox                          │
└─────────────────────────────────────────┘
```

Le GUI et le démon sont **deux binaires séparés** volontairement :
- Le démon reste **pur Go** (portable, cross-compile facile, pas de dépendance système)
- Le GUI est **CGO** (Fyne + OpenGL) mais optionnel (le panneau web suffit)
