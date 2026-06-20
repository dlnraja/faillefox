# 🦊 Faillefox — le pare-feu qui, lui, protège vraiment

> **Faillefox est un VRAI pare-feu gratuit, libre (GPL-3.0) et multiplateforme**
> (Windows / Android / Linux), né en réaction à la parodie
> [`faillefox.com`](https://faillefox.com) — elle-même née d'une perle
> télévisée. Ce dépôt fait l'inverse de la parodie : il construit un outil
> de sécurité **réel**, transparent et open source.

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

**Source vérifiée** :
« Joseph Macé-Scaron alerte sur les dangers de l'intelligence artificielle »,
CNews (YouTube), publié le 16/06/2026, durée 2:46 —
**[youtube.com/watch?v=aZZGPZ4l0_Q](https://www.youtube.com/watch?v=aZZGPZ4l0_Q)**

À la suite de cette séquence, le site **[faillefox.com](https://faillefox.com)**
apparaît : une **parodie** de pare-feu dont le slogan affiché est :

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

## ✨ Que fait Faillefox (v1) ?

Faillefox intercepte les connexions réseau sortantes et vous laisse décider,
**par application**, ce qui a le droit de sortir sur Internet :

- 🟢 **Mode simple** — une liste d'applications avec un interrupteur on/off
  par app (bloquer/autoriser l'accès Internet de chaque programme).
- 🔴 **Mode avancé** — règles précises : application + protocole (TCP/UDP)
  + port + IP.
- 📜 **Journal temps réel** — chaque connexion interceptée et chaque
  décision sont affichées en direct (Server-Sent Events).
- 🎛️ **Politique par défaut** configurable : tout autoriser, tout bloquer,
  ou demander pour chaque nouvelle app.

Le tout pilotable depuis un **panneau web local** ouvert dans votre
navigateur, **jamais exposé sur le réseau**.

### Captures / démonstration

Le binaire actuel tourne avec un **pilote de démonstration (`stub`)** qui
simule des connexions : vous pouvez tester toute l'interface et toute la
logique sans droits administrateur et sans interception réelle. Les pilotes
natifs (WFP, VPNService, nftables) sont en cours d'intégration — voir
[la feuille de route](#-feuille-de-route).

---

## 🚀 Démarrage rapide

### Prérequis
- [Go](https://go.dev/dl/) 1.26+ (pour compiler depuis les sources)

### Compilation & lancement

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
-driver string      pilote de filtrage (stub, windows-wfp, android-vpn, linux-nftables) (défaut "stub")
-port int           port d'écoute du panneau, loopback uniquement (défaut 8443)
-data string        répertoire de données (défaut ~/.faillefox)
-list-drivers       affiche les pilotes compilés et quitte
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
│   • Engine : moteur de décision (rules + default)          │
│   • Journal d'événements (ring buffer + abonnés SSE)       │
│   • Store : persistance JSON (~/.faillefox/policies.json)   │
│   • interface Driver : contrat des backends natifs         │
└───────────────────────┬────────────────────────────────────┘
                        │ interface Driver (Inspect/Apply/ListApps)
        ┌───────────────┼─────────────────────────┐
┌───────▼────────┐ ┌─────▼──────────┐ ┌───────────▼─────────┐
│ windows-wfp    │ │ android-vpn    │ │ linux-nftables      │
│ (WFP callouts, │ │ (VPNService    │ │ (nftables /         │
│  droits admin) │ │  via gomobile) │ │  NFQUEUE/eBPF)      │
└────────────────┘ └────────────────┘ └─────────────────────┘
```

**Principe clé** : `internal/core` ne sait *rien* du filtrage bas niveau.
Il raisonne uniquement en termes d'applications, de règles et de décisions.
C'est ce qui permet de partager **toute** la logique (moteur, journal, API,
UI) entre plateformes — seul le *glue* natif change.

### Organisation du code

| Répertoire | Rôle |
|------------|------|
| `internal/core` | Cœur : types, moteur, règles, journal, store, registry |
| `internal/api` | Serveur HTTP + SSE (loopback) + UI web embarquée |
| `internal/drivers/stub` | Pilote de démonstration (simule des connexions) |
| `cmd/faillefox` | Point d'entrée (`main.go`) |
| `internal/api/web` | UI (HTML/CSS/JS) embarquée via `go:embed` |

---

## 📊 Comparaison avec les projets open source existants

Faillefox se positionne dans un vide : **aucun projet open source majeur
ne couvre aujourd'hui Windows + Android + Linux avec une UI unifiée.**
Chaque projet ci-dessous cible une seule plateforme.

| Projet | Plateformes | Langage | Mécanisme de filtrage | Licence | Lien |
|--------|-------------|---------|-----------------------|---------|------|
| **OpenSnitch** | Linux, macOS | Go + Python + C | eBPF / netfilter, par app | GPL-3.0 | [github.com/opensnitch/opensnitch](https://github.com/opensnitch/opensnitch) |
| **NetGuard** | Android | Java/Kotlin | VPNService, par app | Apache-2.0 | [github.com/M66B/NetGuard](https://github.com/M66B/NetGuard) |
| **simplewall** | Windows | C++ | WFP (Windows Filtering Platform) | GPL-3.0 | [github.com/henrypp/simplewall](https://github.com/henrypp/simplewall) |
| **RethinkDNS** | Android | Kotlin | VPNService + DNS chiffré | MPL-2.0 | [github.com/celzero/rethink-app](https://github.com/celzero/rethink-app) |
| **OpenSnitch-ui** (forks) | Linux | Python | idem OpenSnitch | GPL-3.0 | — |
| **Faillefox** (ce projet) | **Windows + Android + Linux** | **Go** (cœur) | WFP / VPNService / nftables | **GPL-3.0** | ce dépôt |

**Ce qui inspire Faillefox** :
- d'**OpenSnitch** — l'architecture Go + cœur partagé + UI séparée ;
- de **NetGuard** — le modèle de filtrage par application sur Android ;
- de **simplewall** — l'utilisation de WFP pour le filtrage par app sur Windows ;
- de **RethinkDNS** — une UI grand public soignée.

**Ce que Faillefox tente d'apporter en plus** : une UI unique pour tous les
OS (donc un moindre effort de maintenance), et un cœur écrit en Go — un
langage compilé, à la mémoire gérée (sûreté face aux débordements de tampon,
critique pour un outil de sécurité).

---

## 🔒 Considérations de sécurité

- **Canal de contrôle loopback uniquement.** Le serveur HTTP bind sur
  `127.0.0.1`, jamais sur `0.0.0.0`. Le pare-feu ne peut pas être piloté
  depuis le réseau.
- **Persistance en écriture atomique** (fichier temporaire + renommage)
  pour éviter la corruption des règles.
- **Aucune télémétrie, aucun appel réseau sortant du démon lui-même.**
- **Code source ouvert** : tout est auditable. Vous ne faites pas confiance
  à un éditeur, vous lisez le code.
- **Mises en garde honnêtes** : la v1 actuelle utilise un pilote `stub`
  qui ne filtre pas encore réellement le trafic — voir la feuille de route.
  N'utilisez pas la v1 actuelle comme votre seule ligne de défense.

---

## 🗺️ Feuille de route

### v0.1 — Cœur + UI (✅ actuel)
- [x] Cœur Go : moteur de règles, journal, persistance
- [x] API REST + SSE sur loopback
- [x] UI web (mode simple, mode avancé, journal temps réel)
- [x] Pilote `stub` de démonstration (testable sans droits admin)

### v0.2 — Pilote Linux (prochaine étape)
- [ ] Pilote `linux-nftables` via NFQUEUE (interception réelle par app)
- [ ] Détection de l'application émettrice (via `/proc` + socket inode)

### v0.3 — Pilote Windows
- [ ] Pilote `windows-wfp` (callouts WFP en mode utilisateur via `fwpuclnt`)
- [ ] Association PID ↔ connexion (API IP Helper)
- [ ] Service Windows + élévation de privilèges (UAC)

### v0.4 — Pilote Android
- [ ] App Android (Kotlin) enshellant le cœur via `gomobile bind`
- [ ] `VPNService` pour l'interception du trafic par app
- [ ] UI native (Compose) en plus du panneau web

### v1.0 — Stabilisation
- [ ] Tests d'intégration par plateforme
- [ ] Build automatisé (CI GitHub Actions)
- [ ] Documentation d'installation grand public

---

## 🤝 Contribuer

Les contributions sont les bienvenues — **particulièrement** sur les pilotes
natifs (WFP, VPNService, nftables), qui nécessitent une expertise par OS.
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
