# Faillefox pour Linux

Faillefox tourne sur Linux de deux façons :

1. **Manuel** : vous lancez le binaire `faillefox`, le panneau web s'ouvre
   sur http://127.0.0.1:8443.
2. **Service systemd** : Faillefox tourne en arrière-plan comme un service,
   démarré au boot, gérable via `systemctl`.

## Installation via paquet .deb (Debian / Ubuntu / dérivés)

```bash
# Télécharger le paquet depuis la dernière release
# https://github.com/dlnraja/faillefox/releases/latest

sudo dpkg -i faillefox_<version>_amd64.deb
sudo systemctl enable --now faillefox
```

Le paquet installe :
- `/usr/bin/faillefox` — le binaire
- `/etc/systemd/system/faillefox.service` — l'unit systemd (démarrage auto,
  redémarrage en cas de crash, durcissement systemd)
- `/var/lib/faillefox/` — répertoire de données
- Le service démarre avec `CAP_NET_ADMIN` (requis pour `nftables`)

Statut :
```bash
systemctl status faillefox
journalctl -u faillefox -f      # logs en temps réel
```

## Installation manuelle (binaire)

```bash
# Télécharger et décompresser faillefox-linux-amd64.tar.gz
tar xzf faillefox-linux-amd64.tar.gz
sudo ./faillefox-linux-amd64 -driver linux-nftables -dns -cve
```

> Le pilote `linux-nftables` nécessite **root** (ou `CAP_NET_ADMIN`) car il
> manipule `nft`/`iptables`.

## Service systemd manuel (sans le paquet)

```bash
sudo cp deploy/linux/faillefox.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now faillefox
```

## Compilation depuis les sources

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go build -trimpath -ldflags="-s -w" -o faillefox ./cmd/faillefox
sudo ./faillefox -driver linux-nftables -dns -cve
```

## Build du paquet .deb

```bash
./deploy/linux/build-deb.sh        # produit dist/faillefox_<version>_amd64.deb
```

## Désinstallation

```bash
# Paquet
sudo dpkg -r faillefox

# Manuel
sudo systemctl disable --now faillefox
sudo rm /etc/systemd/system/faillefox.service
sudo rm -rf /var/lib/faillefox
```
