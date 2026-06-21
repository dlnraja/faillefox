# Faillefox pour macOS

> ⚠️ **État v0.6** : sur macOS, Faillefox fonctionne en mode **stub** (pas
> d'interception réelle du trafic). Le filtrage temps réel sur macOS
> nécessite l'API **Network Extension** d'Apple, qui demande un compte
> développeur Apple **payant** (99 $/an) + une entitlement `com.apple.developer.networking.networkextension`.
> C'est documenté honnêtement ci-dessous.

## Ce qui fonctionne aujourd'hui (macOS, stub)

- ✅ Cœur complet : moteur de règles, journal, profils
- ✅ DNS sinkhole (résolveur local)
- ✅ Veille CVE (base NVD)
- ✅ Scanner ClamAV (installable via Homebrew)
- ✅ Auto-update des listes
- ✅ Panneau web sur http://127.0.0.1:8443
- ✅ Service launchd (démarrage au boot)

## Ce qui ne fonctionne PAS encore (honnêtement)

- ❌ **Interception réelle du trafic** par application. Sur macOS, la seule
  API officielle est **Network Extension** (PacketTunnelProvider), qui
  nécessite :
  - Un compte développeur Apple **payant** (99 $/an)
  - Une demande d'entitlement NetworkExtension approuvée par Apple
  - Du code natif Swift/Objective-C (pas Go pur)
  - Signature du binaire avec certificat Apple Developer
- En attendant, le DNS sinkhole fonctionne (blocage au niveau DNS pour
  toutes les apps), ce qui couvre déjà une grande partie du besoin.

## Installation

### Via Homebrew (ClamAV optionnel)
```bash
brew install clamav          # pour le scanner ClamAV (optionnel)
```

### Manuel
```bash
# Télécharger faillefox-darwin-arm64.tar.gz (Apple Silicon) ou amd64 (Intel)
tar xzf faillefox-darwin-*.tar.gz
sudo ./faillefox -dns -cve -clamav
```

## Service launchd (démarrage automatique au boot)

```bash
sudo cp deploy/macos/com.dlnraja.faillefox.plist /Library/LaunchDaemons/
sudo launchctl load -w /Library/LaunchDaemons/com.dlnraja.faillefox.plist
```

Logs : `/usr/local/var/log/faillefox.log`

Désactivation :
```bash
sudo launchctl unload /Library/LaunchDaemons/com.dlnraja.faillefox.plist
```

## Compilation depuis les sources

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go build -trimpath -ldflags="-s -w" -o faillefox ./cmd/faillefox
sudo ./faillefox -dns -cve
```

## Feuille de route macOS

Le pilote `darwin-network-extension` est planifié mais nécessite les
prérequis Apple (compte développeur + entitlement). En attendant, le DNS
sinkhole couvre déjà la majorité des cas d'usage (blocage pubs/trackers/
malwares pour toutes les apps).
