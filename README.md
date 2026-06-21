# 🦊 Faillefox — le pare-feu qui, lui, protège vraiment

[![CI](https://github.com/dlnraja/faillefox/actions/workflows/ci.yml/badge.svg)](https://github.com/dlnraja/faillefox/actions/workflows/ci.yml)
[![Lint](https://github.com/dlnraja/faillefox/actions/workflows/lint.yml/badge.svg)](https://github.com/dlnraja/faillefox/actions/workflows/lint.yml)
[![Release](https://github.com/dlnraja/faillefox/actions/workflows/release.yml/badge.svg)](https://github.com/dlnraja/faillefox/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/dlnraja/faillefox)](https://goreportcard.com/report/github.com/dlnraja/faillefox)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/plateformes-Windows%20%C2%B7%20Linux%20%C2%B7%20macOS%20%C2%B7%20Android-success)](#-feuille-de-route)

> **Faillefox est un VRAI pare-feu gratuit, libre (GPL-3.0) et multiplateforme**
> (Windows / Android / Linux), né en réaction à la **parodie**
> [`faillefox.com`](https://faillefox.com) — elle-même née d'une perle
> télévisée. Ce dépôt fait l'inverse de la parodie : il construit un outil
> de sécurité **réel**, transparent et open source.

📖 **Documentation** : [Présentation](docs/presentation.md) · [Architecture](docs/design.md) ·
[Android](docs/android.md) · [Antivirus & signature](docs/antivirus.md) ·
[Feuille de route](ROADMAP.md)

---

## 📺 D'où vient le nom « Faillefox » ?

Le **16 juin 2026**, sur **CNews**, l'essayiste **Joseph Macé-Scaron** —
invité pour parler des dangers de l'intelligence artificielle — s'embrouille
en direct à plusieurs reprises. Il cite des entités aux noms approximatifs
(« *Anthropique* » pour Anthropic ; des logiciels inventés « *Fable 5 et
Mythos 5* »), puis glisse vers Firefox :

> *« Fox, on a détecté de près de 300 failles dans Fox […] C'est le parfeu.
> C'est le parfeu pour tout… pardonnez-moi, c'est un parfeu pour tout. »*

Lui-même reconnaît son embrouille à l'antenne.

**Source vérifiée** : « Joseph Macé-Scaron alerte sur les dangers de
l'intelligence artificielle », CNews (YouTube), 16/06/2026 —
**[youtube.com/watch?v=aZZGPZ4l0_Q](https://www.youtube.com/watch?v=aZZGPZ4l0_Q)**

À la suite de cette séquence, le site **[faillefox.com](https://faillefox.com)**
apparaît : une **parodie** de pare-feu dont le slogan est :

> *« Pare-feu, navigateur, antivirus, IA : Faillefox fait tout. **Sauf vous
> protéger.** »* — accompagné de mentions comme « *461 failles incluses* »
> et « *certifié 0 % sécurisé* ».

**Ce dépôt prend l'idée à contre-pied.** Puisqu'un « pare-feu pour tout »
existe en blague, pourquoi n'existerait-il pas **pour de vrai**, en libre,
multiplateforme, et qui protège effectivement ?

> ⚠️ **Avertissement** : ce projet n'est pas affilié à CNews, à M. Macé-Scaron,
> ni au site parodique faillefox.com. Le nom est utilisé comme clin d'œil
> contextuel. Les propos cités sont publics, diffusés à l'antenne et reconnus
> par leur auteur.

---

## ✨ Que fait Faillefox ?

Faillefox intercepte les connexions réseau sortantes et vous laisse décider,
**par application**, ce qui a le droit de sortir sur Internet :

- 🟢 **Mode simple** — une liste d'applications avec un interrupteur on/off
  par app (bloquer/autoriser l'accès Internet de chaque programme).
- 🔴 **Mode avancé** — règles précises : application + protocole (TCP/UDP)
  + port + IP.
- 🚫 **Blocklist anti-trackers/publicités** optionnelle (façon Pi-hole local).
- 🏠 **Profils réseau** (Maison / Bureau / Public) — blocage plus strict
  automatiquement sur les réseaux publics.
- 📜 **Journal temps réel** — chaque connexion interceptée et chaque
  décision sont affichées en direct (SSE) **et** persistées sur disque
  (rotation automatique).
- 🎛️ **Politique par défaut** configurable : tout autoriser, tout bloquer,
  ou demander pour chaque nouvelle app.

Le tout pilotable depuis un **panneau web local** ouvert dans votre
navigateur, **jamais exposé sur le réseau**.

### Plateformes supportées (v0.2)

| Plateforme | Pilote | Statut |
|------------|--------|--------|
| **Windows** | `windows-netfw` (Pare-feu Windows via `netsh`) | ✅ réel |
| **Linux** | `linux-nftables` (nftables / iptables) | ✅ réel |
| **macOS** | `stub` | ⏳ stub (v0.5) |
| **Android** | `android-vpn` (VPNService + gomobile) | ⏳ scaffold (v0.4) |
| Toutes | `stub` (simulation) | ✅ démo sans droits admin |

---

## 🚀 Démarrage rapide

### Télécharger les binaires

👉 **[Dernière release](https://github.com/dlnraja/faillefox/releases/latest)**
binaires Windows / Linux / macOS (amd64 + arm64) + sommes SHA256.

### Compiler depuis les sources

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go build -o faillefox ./cmd/faillefox
./faillefox
```

Puis ouvrez **http://127.0.0.1:8443** dans votre navigateur.

> Le panneau de contrôle n'écoute **que** sur `127.0.0.1`. Il n'est jamais
> accessible depuis une autre machine. C'est une règle de sécurité
> non négociable pour un pare-feu : son canal de contrôle ne doit pas être
> joignable depuis le réseau.

### Options en ligne de commande

```text
-driver string      pilote: windows-netfw | linux-nftables | stub (défaut auto)
-port int           port d'écoute du panneau, loopback uniquement (défaut 8443)
-data string        répertoire de données (défaut ~/.faillefox)
-profile string     profil réseau: home | office | public (défaut home)
-blocklist string   fichier hosts à charger comme liste anti-trackers
-no-persistent-log  désactive le journal persistant sur disque
-list-drivers       affiche les pilotes compilés et quitte
```

### Exemples

```bash
# Démarrage normal (pilote auto selon l'OS)
./faillefox

# Profil public + blocklist anti-pubs
./faillefox -profile public -blocklist blocklist.txt

# Windows : pilote Pare-feu Windows (nécessite droits admin pour netsh)
./faillefox.exe -driver windows-netfw

# Linux : pilote nftables (nécessite root)
sudo ./faillefox -driver linux-nftables
```

---

## 🏗️ Architecture

```
┌────────────────────────────────────────────────────────────┐
│              UI WEB (HTML/CSS/JS vanilla, embarquée)        │
│   Mode Simple (interrupteurs)  │  Mode Avancé (règles)      │
│   Journal temps réel (SSE)     │  servie par le démon       │
└───────────────────────┬────────────────────────────────────┘
                        │ HTTP REST + SSE (loopback 127.0.0.1)
┌───────────────────────▼────────────────────────────────────┐
│            internal/core  —  cœur Go partagé                │
│   • Engine : moteur de décision (rules + default + blocklist)│
│   • Profils réseau (home/office/public)                     │
│   • Journal d'événements (ring buffer + abonnés SSE)       │
│   • Store : persistance JSON (~/.faillefox/policies.json)   │
│   • interface Driver : contrat des backends natifs         │
└───────────────────────┬────────────────────────────────────┘
                        │ interface Driver (Inspect/Apply/ListApps)
        ┌───────────────┼────────────────┬───────────────────┐
┌───────▼────────┐ ┌─────▼──────────┐ ┌──▼──────────┐ ┌─────▼────────┐
│ windows-netfw  │ │ linux-nftables │ │ android-vpn │ │ stub (démo)  │
│ Pare-feu Win   │ │ nftables/      │ │ VPNService  │ │ simulation   │
│ (netsh)        │ │ iptables       │ │ + gomobile  │ │              │
└────────────────┘ └────────────────┘ └─────────────┘ └──────────────┘
        │                  │                │
   droits admin       root/CAP_NET_ADMIN   autorisation VPN
```

**Principe clé** : `internal/core` ne sait *rien* du filtrage bas niveau.
Il raisonne uniquement en termes d'applications, de règles et de décisions.
C'est ce qui permet de partager **toute** la logique (moteur, journal, API,
UI) entre plateformes — seul le *glue* natif change.

### Organisation du code

| Répertoire | Rôle |
|------------|------|
| `internal/core` | Cœur : types, moteur, règles, journal, store, blocklist, profils, registry |
| `internal/api` | Serveur HTTP + SSE (loopback) + UI web embarquée |
| `internal/logging` | Journal persistant rotatif (JSONL) |
| `internal/drivers/stub` | Pilote de démonstration |
| `internal/drivers/netfw` | Pilote Windows (Pare-feu Windows) |
| `internal/drivers/nftables` | Pilote Linux (nftables/iptables) |
| `pkg/android` | Bindings gomobile pour l'app Android |
| `android/` | App Android Kotlin (Gradle + VpnService) |
| `cmd/faillefox` | Point d'entrée (`main.go`) |

---

## 📊 Comparaison avec les projets open source existants

Faillefox se positionne dans un vide : **aucun projet open source majeur
ne couvre aujourd'hui Windows + Android + Linux avec une UI unifiée.**

| Projet | Plateformes | Langage | Mécanisme | Licence |
|--------|-------------|---------|-----------|---------|
| [OpenSnitch](https://github.com/opensnitch/opensnitch) | Linux, macOS | Go + Python | eBPF / netfilter | GPL-3.0 |
| [NetGuard](https://github.com/M66B/NetGuard) | Android | Kotlin | VPNService | Apache-2.0 |
| [simplewall](https://github.com/henrypp/simplewall) | Windows | C++ | WFP | GPL-3.0 |
| [RethinkDNS](https://github.com/celzero/rethink-app) | Android | Kotlin | VPNService + DNS | MPL-2.0 |
| **Faillefox** | **Windows + Android + Linux** | **Go** | netsh / nftables / VpnService | **GPL-3.0** |

**Ce qui inspire Faillefox** :
- d'**OpenSnitch** — l'architecture Go + cœur partagé + UI séparée ;
- de **NetGuard** — le filtrage par application sur Android ;
- de **simplewall** — l'utilisation de l'API pare-feu Windows ;
- de **RethinkDNS** — une UI grand public soignée + DNS.

---

## 🔒 Considérations de sécurité & antivirus

- **Canal de contrôle loopback uniquement.** Le serveur HTTP bind sur
  `127.0.0.1`, jamais sur `0.0.0.0`.
- **Persistance en écriture atomique** (fichier temporaire + renommage).
- **Aucune télémétrie, aucun appel réseau sortant du démon lui-même.**
- **Code source ouvert** (GPL-3.0) : tout est auditable.
- **Sûreté mémoire** : cœur en Go (GC, pas de débordements de tampon).
- **Signature Authenticode** via SignPath (workflow prêt) — voir
  [docs/antivirus.md](docs/antivirus.md).
- **Honnêteté** : la v0.2 filtre réellement via `netsh`/`nftables` mais
  n'est pas un pare-feu noyau complet — c'est dit clairement, pour ne pas
  créer de faux sentiment de sécurité.

📝 **Faux positifs antivirus ?** Faillefox écoute un port local et pilote
le pare-feu système : ce sont des actions que les heuristiques AV peuvent
trouver suspectes. La procédure complète (signature SignPath gratuite +
soumission aux 10 principaux labs antivirus) est documentée dans
**[docs/antivirus.md](docs/antivirus.md)**.

---

## 🗺️ Feuille de route

### ✅ v0.1 — Cœur + UI + pilote stub
Cœur Go, API REST + SSE, UI web, pilote stub, CI multi-OS, release.

### ✅ v0.2 — Vrais pilotes + anti-trackers + Android (actuelle)
- Pilotes Windows (`windows-netfw`) et Linux (`linux-nftables`) réels
- Blocklist anti-trackers, profils réseau, journal persistant rotatif
- Scaffold Android complet (VpnService + Kotlin + gomobile)
- Workflow SignPath + guide antivirus

### 🔜 v0.3 — Pilote WFP avancé + filtrage strict par app
Callouts WFP mode utilisateur, association PID↔connexion, mode `ask` prompt.

### 🔜 v0.4 — Android complet
Forward tun2socks, filtrage par UID, UI Compose détaillée, F-Droid.

### 🔜 v1.0 — Stabilisation & grand public
Installateurs natifs, signature auto, doc grand public, revue sécurité.

Voir [`ROADMAP.md`](ROADMAP.md) pour le détail complet.

---

## 🧪 Qualité & CI

- **23 tests unitaires** (moteur, règles, journal, store, blocklist, profils,
  logger rotatif) — `go test ./...`
- **CI** sur matrice Ubuntu / Windows / macOS (build + vet + test).
- **Lint** golangci-lint v2.12 (compatible Go 1.26).
- **Release** multi-plateforme automatique sur tag.

---

## 🤝 Contribuer

Les contributions sont les bienvenues — **particulièrement** sur les pilotes
natifs (WFP, VPNService, nftables) qui nécessitent une expertise par OS.
Voir [`CONTRIBUTING.md`](CONTRIBUTING.md).

---

## 📄 Licence

**GNU General Public License v3.0** — voir [`LICENSE`](LICENSE).

Pourquoi la GPL ? Parce qu'un outil de sécurité doit rester auditable et
libre pour tous. Toute redistribution doit conserver ces garanties.

---

## ❤️ Remerciements

- À **[faillefox.com](https://faillefox.com)** pour l'idée — involontairement.
- Aux projets **OpenSnitch**, **NetGuard**, **simplewall** et **RethinkDNS**
  qui ouvrent la voie du filtrage par application open source.
