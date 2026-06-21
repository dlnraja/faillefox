# Faillefox pour Windows

Faillefox s'exécute sur Windows de deux façons :

1. **Application console** (par défaut) — vous lancez `faillefox.exe`, le
   panneau web s'ouvre sur http://127.0.0.1:8443.
2. **Service Windows natif** — Faillefox tourne en arrière-plan comme un
   service Windows, démarré automatiquement au boot, gérable via
   `services.msc`.

## Installation rapide

Téléchargez la dernière release Windows :
https://github.com/dlnraja/faillefox/releases/latest

Décompressez `faillefox-windows-amd64.exe.zip`, puis :

```cmd
:: Lancement console (panneau web sur http://127.0.0.1:8443)
faillefox.exe

:: Tout activer : pare-feu + DNS + CVE + ClamAV + auto-update
faillefox.exe -driver windows-netfw -dns -cve -clamav -freshclam
```

> ⚠️ Le pilote `windows-netfw` (Pare-feu Windows via `netsh`) nécessite
> les droits administrateur. Cliquez droit → « Exécuter en tant
> qu'administrateur » ou installez-le comme service (ci-dessous).

## Service Windows natif (démarrage automatique au boot)

```cmd
:: 1. Ouvrir une console EN TANT QU'ADMINISTRATEUR
:: 2. Installer le service
faillefox.exe -winsvc install

:: 3. Démarrer le service
faillefox.exe -winsvc start
```

À partir de là, Faillefox démarre **automatiquement à chaque boot**, avant
même qu'un utilisateur ne se connecte. Gérable via :
- **`services.msc`** → service « Faillefox »
- **`net start Faillefox`** / **`net stop Faillefox`**
- **`faillefox.exe -winsvc stop`** / **`-winsvc uninstall`**

## Options Windows utiles

| Option | Effet |
|--------|-------|
| `-driver windows-netfw` | Pilote Pare-feu Windows (filtrage par app) |
| `-winsvc install` | Installer comme service Windows |
| `-winsvc start\|stop` | Démarrer / arrêter le service |
| `-winsvc uninstall` | Désinstaller le service |
| `-port 8443` | Port du panneau web (loopback) |

## Faux positifs antivirus

Faillefox écoute un port local et pilote le Pare-feu Windows : ces actions
peuvent déclencher un faux positif antivirus. Voir
[docs/antivirus.md](../../docs/antivirus.md) pour la signature SignPath et la
soumission aux labs AV. Les binaires de release incluent les métadonnées
VERSIONINFO (Company, Product, Version) + manifeste UAC `asInvoker` pour
réduire ces faux positifs.

## Désinstallation

```cmd
faillefox.exe -winsvc uninstall   :: si installé comme service
del faillefox.exe                 :: supprimer le binaire
rmdir /s %USERPROFILE%\.faillefox  :: supprimer les données
```

## Compilation depuis les sources

```cmd
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go build -trimpath -ldflags="-s -w" -o faillefox.exe ./cmd/faillefox
```

Le code spécifique Windows (service) est dans
`internal/platform/winsvc/winsvc.go` (build tag `//go:build windows`).
