# Changelog

Tous les changements notables de Faillefox sont documentés ici.
Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/lang/fr/).

## [Non publié]

Rien pour l'instant.## [v0.7.0] - 2026-06-21

### Ajouté

### Corrigé

## [v0.7.1] - 2026-06-21

### Ajouté

### Corrigé


## [v0.6.0] - 2026-06-21

### Ajouté

### Corrigé


## [v0.5.0] — 2026-06-21

### Ajouté
- **UI web complète** : les modules v0.3/v0.4 (DNS sinkhole, veille CVE,
  scanner ClamAV, auto-update) sont enfin visibles dans le tableau de bord.
- 3 nouveaux endpoints API : `GET /api/updater`, `GET|POST /api/cve`,
  `GET /api/scan`.
- 3 nouveaux onglets UI : **Auto-update** (statut détaillé du rafraîchissement,
  rafraîchi toutes les 30 s), **CVE** (formulaire nom+version → alertes),
  **Scan ClamAV** (formulaire chemin → résultat).
- CSS : styles `.kv` (blocs statut) et `.alerts` (cartes CVE avec code
  couleur par sévérité CRITICAL/HIGH/MEDIUM/LOW).

### Technique
- `internal/api/server.go` : setters optionnels `SetUpdater`/`SetScanner`/`SetFeed`.
- `cmd/faillefox/main.go` : récupération des refs modules et branchement au serveur.

## [v0.4.0] — 2026-06-21

### Ajouté
- **Auto-update activé par défaut** : listes DNS (StevenBlack, OISD) + base CVE
  téléchargées au démarrage, puis rafraîchies toutes les **6 h** en arrière-plan.
- **Signatures ClamAV auto** : intégration de `freshclam` (option `-freshclam`),
  mise à jour toutes les 2 h.
- **Workflow auto-release hebdomadaire** (`weekly-release.yml`) : un tag
  auto-daté (`v0.AAAA.SEMAINE.SEQ`) est créé chaque lundi et déclenche une
  release multi-OS automatiquement via `workflow_call`.
- **Dependabot** : PR automatiques pour les dépendances Go (`miekg/dns`) et
  les GitHub Actions.
- Observabilité `updater.Status()` (dernier fetch, nombre de domaines, cycle).
- Nouveaux flags CLI : `-no-autoupdate`, `-update-every`, `-freshclam`.

### Technique
- `internal/freshclam` : nouveau module d'auto-update des signatures ClamAV.
- `internal/updater` : intervalle par défaut 6 h (was 24 h), état observable.

## [v0.3.0] — 2026-06-21

### Ajouté
- **DNS sinkhole** (`internal/dnsshield`) : résolveur DNS local qui bloque
  pubs/trackers/malwares pour tout le système (façon Pi-hole). Upstreams
  1.1.1.1, 9.9.9.9, 8.8.8.8. Dépendance `github.com/miekg/dns`.
- **Veille CVE** (`internal/cvefeed`) : interroge la base NVD officielle
  (gratuite, publique) et alerte si un logiciel installé a une faille connue.
- **Scanner ClamAV** (`internal/clamscan`) : intégration du seul moteur AV
  open source via `clamd` (daemon) et `clamscan` (CLI). Documenté comme
  limité vs solutions commerciales.
- **Auto-update des listes DNS** (`internal/updater`) : sources StevenBlack,
  OISD, Abuse.ch.
- Nouveaux flags CLI : `-dns`, `-dns-port`, `-cve`, `-clamav`.
- `docs/clamav.md` : guide d'installation + tableau comparatif honnête
  ClamAV vs Kaspersky.

### Technique
- Dépendance externe ajoutée : `github.com/miekg/dns v1.1.72`.
- Note de périmètre ajoutée au README : Faillefox est un bouclier réseau/DNS
  + veille CVE + scan, **pas** un antivirus temps réel.

## [v0.2.0] — 2026-06-21

### Ajouté
- **Pilote Windows réel** `windows-netfw` : Pare-feu Windows via
  `netsh advfirewall` (filtrage par application, droits admin requis).
- **Pilote Linux réel** `linux-nftables` : nftables / iptables.
- **Blocklist anti-trackers/publicités** (façon Pi-hole local) dans `internal/core`.
- **Profils réseau** (Maison / Bureau / Public) avec politique par défaut conseillée.
- **Journal persistant rotatif** (JSONL sur disque) dans `internal/logging`.
- **Scaffold Android complet** : `pkg/android` (bindings gomobile), `android/`
  (app Kotlin avec Gradle + VpnService + MainActivity + UI).
- **Workflow SignPath** (signature Authenticode gratuite pour OSS).
- **Guide antivirus** `docs/antivirus.md` : procédure de soumission aux 10
  principaux labs antivirus.
- Métadonnées `version_info.json` + manifeste UAC `asInvoker` Windows.
- `docs/presentation.md`, `docs/android.md`, `ROADMAP.md`.

## [v0.1.0] — 2026-06-20

### Ajouté
- Cœur Go : moteur de règles, journal en mémoire, persistance JSON,
  interface `Driver`, registry.
- API REST + SSE sur loopback (`127.0.0.1`).
- UI web (mode simple interrupteurs, mode avancé règles, journal temps réel).
- Pilote `stub` de démonstration (simule des connexions).
- 14 tests unitaires.
- CI GitHub Actions sur matrice Ubuntu / Windows / macOS (build + vet + test).
- Lint golangci-lint v2.12 (compatible Go 1.26).
- Release multi-plateforme automatique (6 binaires + SHA256SUMS).
- README avec l'histoire vérifiée (CNews 16/06/2026 + source YouTube) et
  comparatif aux projets OSS (OpenSnitch, NetGuard, simplewall, RethinkDNS).

[Non publié]: https://github.com/dlnraja/faillefox/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/dlnraja/faillefox/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/dlnraja/faillefox/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/dlnraja/faillefox/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/dlnraja/faillefox/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/dlnraja/faillefox/releases/tag/v0.1.0
