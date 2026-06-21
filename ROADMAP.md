# Feuille de route Faillefox

Ce document trace les versions passées et à venir. Chaque version ajoute
une couche de fonctionnalité tout en gardant le cœur stable.

---

## ✅ v0.1 — Cœur + UI + pilote stub (publiée le 20/06/2026)

- [x] Cœur Go : moteur de règles, journal mémoire, persistance JSON
- [x] API REST + SSE sur loopback (`127.0.0.1`)
- [x] UI web (mode simple / mode avancé / journal temps réel)
- [x] Pilote `stub` de démonstration
- [x] 14 tests unitaires
- [x] CI multi-OS (Ubuntu / Windows / macOS)
- [x] Release multi-plateforme (6 binaires + SHA256SUMS)

## ✅ v0.2 — Vrais pilotes + anti-trackers + Android (publiée le 21/06/2026)

- [x] Pilote Windows réel `windows-netfw` (Pare-feu Windows via `netsh`)
- [x] Pilote Linux réel `linux-nftables` (nftables / iptables)
- [x] Blocklist anti-trackers (façon Pi-hole local)
- [x] Profils réseau (Maison / Bureau / Public)
- [x] Journal persistant rotatif (JSONL sur disque)
- [x] Scaffold Android complet (Gradle + Kotlin + VpnService + gomobile)
- [x] Workflow SignPath (signature Authenticode gratuite OSS)
- [x] Guide antivirus (soumission aux labs)
- [x] Métadonnées VERSIONINFO + manifeste UAC Windows

## ✅ v0.3 — DNS sinkhole + CVE + ClamAV (publiée le 21/06/2026)

Bouclier réseau/DNS complet + veille vulnérabilités + scan à la demande.
**Honnêtement pas un antivirus temps réel** (voir docs/clamav.md) — c'est
un complément qui s'ajoute à Windows Defender ou un AV commercial.

- [x] **DNS sinkhole** : résolveur local 127.0.0.1 qui bloque pubs/trackers/
      malwares pour tout le système (façon Pi-hole)
- [x] **Auto-update** des listes DNS (sources StevenBlack, OISD, Abuse.ch)
- [x] **Veille CVE** : interroge la base NVD officielle (gratuite) et alerte
      si un logiciel installé a une faille connue
- [x] **Scanner ClamAV** : seul moteur AV open source, intégré via clamd
      (daemon) et clamscan (CLI) — limité vs solutions commerciales
- [x] Dépendance `miekg/dns` (librairie Go de référence pour DNS)
- [x] ~17 nouveaux tests (DNS sinkhole, CVE feed, ClamAV parser, updater)

## 🔜 v0.4 — Android complet + UI de scan

- [ ] Forward des paquets via tun2socks (interception réelle sur Android)
- [ ] Filtrage par UID (par app Android)
- [ ] UI native détaillée (Jetpack Compose) : liste des apps, journal
- [ ] UI de scan ClamAV dans le panneau web (fichiers/dossiers)
- [ ] Notifications système pour le mode `ask` et les CVE
- [ ] Publication sur F-Droid (apks signés reproductiblement)

## 🔜 v0.5 — Pilote WFP avancé + filtrage strict par app

Objectif : filtrage par application **strict** et en temps réel sur Windows,
via WFP (Windows Filtering Platform).

- [ ] Pilote `windows-wfp` (callouts WFP en mode utilisateur via `fwpuclnt`)
- [ ] Association PID ↔ connexion (`GetExtendedTcpTable`)
- [ ] Service Windows + élévation UAC automatique
- [ ] Mode `ask` : prompt système à la première connexion d'une app
- [ ] Pilote Linux : association paquet→PID via NFQUEUE + `/proc/net`
- [ ] Tests d'intégration par plateforme

## 🔜 v0.5 — Fonctionnalités avancées

- [ ] Règles par géolocalisation IP (bloquer un pays)
- [ ] Listes de blocage communautaires (StevenBlack, EasyList…)
- [ ] Statistiques d'usage par application (volume, fréquence)
- [ ] Export / import des règles (JSON)
- [ ] Mode « verrouillage » (bloquer tout nouveau logiciel réseau)

## 🔜 v1.0 — Stabilisation & grand public

- [ ] Installateurs natifs (.msi Windows, .deb/.rpm Linux)
- [ ] Signature automatique SignPath activée sur chaque release
- [ ] Documentation d'installation grand public (avec captures)
- [ ] Page de téléchargement claire
- [ ] Revue de sécurité externe
- [ ] Programme de divulgation responsable des vulnérabilités

---

## Principes directeurs (non négociables)

1. **Loopback uniquement** pour le canal de contrôle. Jamais exposé au réseau.
2. **Aucune télémétrie**, aucun appel réseau sortant du démon.
3. **Code source ouvert** (GPL-3.0), auditable.
4. **Honnêteté** : on ne prétend jamais une protection qu'on n'apporte pas.
   (La v0.2 filtre réellement via netsh/nftables mais n'est pas un pare-feu
   noyau complet — c'est dit clairement.)
5. **Sûreté mémoire** : cœur en Go (GC, pas de débordements de tampon).
