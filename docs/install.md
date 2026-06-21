# Installation de Faillefox

Guide d'installation par plateforme. Choisissez la méthode qui correspond à
votre OS et à votre niveau de confort.

> 👉 **Le plus simple** : téléchargez un installateur/binaire tout prêt
> depuis la [page des releases](https://github.com/dlnraja/faillefox/releases/latest).

---

## 🪟 Windows

### Méthode 1 — Installateur `.exe` (le plus simple)

1. Téléchargez **`faillefox-setup-<version>.exe`** depuis la
   [dernière release](https://github.com/dlnraja/faillefox/releases/latest).
2. Double-cliquez dessus (droits admin requis).
3. Suivez l'assistant **Suivant / Suivant / Terminer** :
   - raccourci Bureau (optionnel)
   - **service Windows au démarrage** (recommandé — démarrage auto au boot)
4. C'est tout. Le panneau web est sur **http://127.0.0.1:8443**.

**Désinstallation** : « Paramètres → Applications → Faillefox → Désinstaller »
(ou lancez `C:\Program Files\Faillefox\unins000.exe`).

### Méthode 2 — Binaire portable (sans installateur)

1. Téléchargez **`faillefox-windows-amd64.exe.zip`**.
2. Décompressez où vous voulez (ex: `C:\Tools\Faillefox\`).
3. Ouvrez une console **en tant qu'administrateur** :
   ```cmd
   faillefox.exe -driver windows-netfw -dns -cve -clamav
   ```
4. Pour l'auto-démarrage sans installateur :
   ```cmd
   faillefox.exe -winsvc install
   faillefox.exe -winsvc start
   ```

📖 Détails : [deploy/windows/README.md](../deploy/windows/README.md)

---

## 🐧 Linux

### Debian / Ubuntu / Mint (paquet `.deb`)

```bash
# Téléchargez faillefox_<version>_amd64.deb depuis la release
sudo dpkg -i faillefox_*_amd64.deb
sudo systemctl enable --now faillefox   # démarre au boot
```

Statut : `systemctl status faillefox` · Logs : `journalctl -u faillefox -f`

### Fedora / RHEL / openSUSE (paquet `.rpm`)

```bash
sudo dnf install faillefox-*.rpm        # Fedora/RHEL
# ou : sudo zypper install faillefox-*.rpm   # openSUSE
sudo systemctl enable --now faillefox
```

### Arch Linux (AUR — bientôt)

```bash
yay -S faillefox     # quand le paquet AUR sera soumis
```

### Installation manuelle (toutes distros)

```bash
tar xzf faillefox-linux-amd64.tar.gz
sudo ./faillefox -driver linux-nftables -dns -cve -threat-intel
# Ou via systemd :
sudo cp deploy/linux/faillefox.service /etc/systemd/system/
sudo systemctl enable --now faillefox
```

📖 Détails : [deploy/linux/README.md](../deploy/linux/README.md)

---

## 🍎 macOS

### Homebrew (le plus simple)

```bash
brew tap dlnraja/faillefox https://github.com/dlnraja/faillefox
brew install faillefox
brew services start faillefox     # démarre au boot via launchd
```

### Installation manuelle

```bash
# Apple Silicon (M1/M2/M3) : faillefox-darwin-arm64.tar.gz
# Intel : faillefox-darwin-amd64.tar.gz
tar xzf faillefox-darwin-*.tar.gz
sudo ./faillefox -dns -cve -threat-intel

# Démarrage au boot via launchd :
sudo cp deploy/macos/com.dlnraja.faillefox.plist /Library/LaunchDaemons/
sudo launchctl load -w /Library/LaunchDaemons/com.dlnraja.faillefox.plist
```

> ⚠️ Sur macOS, le filtrage temps réel par application nécessite l'API
> **Network Extension** d'Apple (compte développeur payant). Le DNS sinkhole
> et la veille CVE fonctionnent dès maintenant. Voir
> [deploy/macos/README.md](../deploy/macos/README.md).

---

## 📱 Android

### APK (sideloading)

1. Téléchargez **`faillefox-<version>.apk`** depuis la release.
2. Activez « Sources inconnues » dans les paramètres Android.
3. Ouvrez l'APK.
4. Appuyez sur **« Activer le pare-feu »** — Android demandera l'autorisation
   VPN (le filtrage réseau passe par un VPN local, cf. NetGuard).

> ⚠️ Faillefox Android est en **scaffold** (v0.7). Le tunnel VPN est
> implémenté mais le forward complet des paquets (tun2socks) arrive en v0.8.
> Publication F-Droid prévue en v0.8.

📖 Détails : [docs/android.md](android.md)

---

## 🛠️ Compilation depuis les sources (toutes plateformes)

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go build -trimpath -ldflags="-s -w" -o faillefox ./cmd/faillefox
./faillefox
```

Voir [docs/build.md](build.md) pour la cross-compilation, le VERSIONINFO
Windows, et le build de l'APK Android.

---

## Vérification de l'installation

Après installation, ouvrez **http://127.0.0.1:8443** dans votre navigateur.
Vous devriez voir le tableau de bord Faillefox avec ses 6 onglets.

Vérifiez que le service tourne :
- **Windows** : `services.msc` → service « Faillefox »
- **Linux** : `systemctl status faillefox`
- **macOS** : `brew services info faillefox` ou `launchctl list | grep faillefox`

## Démarrage rapide après installation

```bash
# Tout activer (pare-feu + DNS + CVE + threat intel + ClamAV)
faillefox -dns -cve -threat-intel -clamav -freshclam

# Puis ouvrez http://127.0.0.1:8443
```

## Désinstallation

| OS | Procédure |
|----|-----------|
| Windows (installateur) | Panneau de config → Désinstaller Faillefox (arrête et supprime le service automatiquement) |
| Windows (manuel) | `faillefox.exe -winsvc uninstall` puis supprimer le binaire |
| Linux .deb | `sudo dpkg -r faillefox` |
| Linux .rpm | `sudo dnf remove faillefox` / `zypper remove faillefox` |
| macOS Homebrew | `brew uninstall faillefox` |
| macOS manuel | `sudo launchctl unload /Library/LaunchDaemons/com.dlnraja.faillefox.plist` puis supprimer le binaire |
| Toutes | Supprimer `~/.faillefox/` (données, journaux, gamification) |
