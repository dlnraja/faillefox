# Architecture de Faillefox

Document de référence sur l'architecture interne : couches, flux de données,
décisions de conception (ADR), et points d'extension pour contribuer.

## 1. Principes directeurs

1. **Loopback uniquement** pour le canal de contrôle. Non négociable.
2. **Aucune télémétrie**, aucun appel réseau sortant du démon (sauf listes
   publiques explicitement demandées par l'utilisateur).
3. **Sûreté mémoire** : cœur en Go (GC, pas de débordements).
4. **Honnêteté** : on ne prétend jamais une protection qu'on n'apporte pas.
5. **Code auditable** : GPL-3.0, sources ouvertes.

## 2. Couches

```
┌─────────────────────────────────────────────────────────────┐
│  UI WEB (HTML/CSS/JS vanilla, embarquée via go:embed)        │
│  6 onglets : simple · avancé · journal · auto-update ·       │
│              CVE · scan ClamAV                                │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP REST + SSE (loopback 127.0.0.1)
┌──────────────────────────▼──────────────────────────────────┐
│  internal/api — Serveur de contrôle                          │
│  Endpoints: /api/status, /api/rules, /api/events (SSE),      │
│             /api/updater, /api/cve, /api/scan                │
└──────────────────────────┬──────────────────────────────────┘
                           │ appels directs (mémoire)
┌──────────────────────────▼──────────────────────────────────┐
│  internal/core — Cœur métier partagé (sans I/O réseau)       │
│  • Engine : moteur de décision (rules + default + blocklist) │
│  • Blocklist : domaines bloqués (anti-trackers)              │
│  • ProfileManager : profils réseau (home/office/public)      │
│  • Store : persistance JSON atomique                         │
│  • interface Driver : contrat des backends natifs            │
└──────────────────────────┬──────────────────────────────────┘
                           │ interface Driver (Start/ApplyRules/ListApps)
   ┌───────────────────────┼───────────────────────┬──────────┐
   │                       │                       │          │
┌──▼─────────────┐ ┌───────▼────────┐ ┌────────────▼──┐ ┌────▼────────┐
│ windows-netfw  │ │ linux-nftables │ │ android-vpn   │ │ stub (démo) │
│ netsh advfirew │ │ nft / iptables │ │ VPNService    │ │ simulation  │
└────────────────┘ └────────────────┘ └───────────────┘ └─────────────┘

Modules complémentaires (v0.3+) branchés au cœur :

┌─────────────────────────────────────────────────────────────┐
│  internal/dnsshield  — Résolveur DNS sinkhole (miekg/dns)    │
│  internal/updater    — Auto-update listes DNS + CVE (6h)     │
│  internal/cvefeed    — Veille CVE (base NVD officielle)      │
│  internal/clamscan   — Scanner ClamAV (clamd / clamscan)     │
│  internal/freshclam  — MAJ signatures ClamAV (2h)            │
│  internal/logging    — Journal persistant rotatif (JSONL)    │
└─────────────────────────────────────────────────────────────┘
```

## 3. Flux de données : une connexion interceptée

```
1. Une app tente une connexion sortante.
2. Le pilote natif (netfw/nftables) l'intercepte.
3. Le pilote appelle engine.Decide(conn).
4. Le moteur évalue, DANS L'ORDRE :
   a. Blocklist (si domaine bloqué)  -> DENY "blocklist"
   b. Règles utilisateur (1re qui matche) -> son action
   c. Politique par défaut            -> allow/deny/ask
5. Le moteur journalise l'Event :
   - ring buffer mémoire (UI live)
   - sink persistant (events.jsonl rotatif)
   - abonnés SSE (navigateur)
6. Le pilote applique la décision (drop / forward).
```

## 4. Décisions de conception (ADR)

### ADR-001 : Cœur en Go, glue natif par OS
**Décision** : 90 % du code en Go pur (partagé), 10 % de glue spécifique
par plateforme (netsh / nftables / VPNService).
**Pour** : partage maximal, sûreté mémoire, cross-compile facile.
**Contre** : le filtrage strict par app (PID↔paquet) demande des APIs
non-Go (WFP callouts C, /proc parsing) — reporté en v0.6.

### ADR-002 : UI web embarquée plutôt que native
**Décision** : une seule UI HTML/CSS/JS servie par le démon, embarquée
via `go:embed`.
**Pour** : une seule UI à maintenir pour tous les OS, pas de dépendance GUI.
**Contre** : moins « native » qu'une UI WinUI/Compose/GTK. Accepté.

### ADR-003 : API sur loopback uniquement
**Décision** : le serveur HTTP bind sur `127.0.0.1`, jamais `0.0.0.0`.
**Pour** : le canal de contrôle n'est pas joignable depuis le réseau —
critique pour un pare-feu.
**Contre** : pas de pilotage distant. C'est voulu (sécurité d'abord).

### ADR-004 : ClamAV intégré, pas de moteur AV maison
**Décision** : intégrer ClamAV (seul AV open source) pour le scan à la
demande, plutôt que d'écrire un moteur heuristique.
**Pour** : ClamAV est maintenu par Cisco Talos, signatures mises à jour
en continu, gratuit.
**Contre** : ClamAV est limité vs solutions commerciales (pas de ML, pas
de sandbox). Documenté honnêtement dans docs/clamav.md.

### ADR-005 : Auto-update activé par défaut
**Décision** : le démon télécharge les listes DNS + CVE au démarrage puis
toutes les 6h, sans intervention.
**Pour** : l'utilisateur a toujours des listes fraîches (malwares émergents).
**Contre** : appels réseau au démarrage. Désactivable via `-no-autoupdate`.

### ADR-006 : Conventional commits + auto-version
**Décision** : les messages de commit suivent le format conventional
(`feat:`/`fix:`/etc.), un workflow bump la version + CHANGELOG + tag
automatiquement.
**Pour** : releases fréquentes sans charge mentale, historique propre.
**Contre** : discipline de commit. Le PR template rappelle les règles.

## 5. Points d'extension (comment contribuer)

| Envie de… | Fichier à toucher | Interface à respecter |
|-----------|-------------------|----------------------|
| Ajouter un pilote natif | `internal/drivers/<nom>/` | implémenter `core.Driver` + `RegisterDriver` |
| Ajouter une source de blocklist | `internal/updater/updater.go` | ajouter une URL à `dnsSources` |
| Ajouter un endpoint API | `internal/api/server.go` | handler + route dans `New()` |
| Ajouter un onglet UI | `internal/api/web/index.html` + `app.js` | `<button class="tab">` + section panel |
| Ajouter un module (ex: anti-malware URL) | `internal/<nom>/` | package Go + branchement `main.go` |

## 6. Sécurité du dépôt

- `permissions: contents: write` limitée aux workflows qui en ont besoin
  (release, auto-version, weekly-release).
- Aucun secret en clair dans le code. Les secrets SignPath sont des
  GitHub Secrets.
- Dependabot surveille les dépendances (CVE).
- `SECURITY.md` documente la divulgation responsable.

## 7. Limitations connues (v0.5)

- **Pilote Windows** : filtrage par app via `netsh` (règles statiques), pas
  encore d'interception temps réel par PID (WFP callouts — v0.6).
- **Pilote Linux** : blocage par destination (IP/port), pas encore par PID
  strict (NFQUEUE — v0.6).
- **Android** : scaffold VPN en place, forward des paquets pas encore implémenté
  (tun2socks — v0.5 Android).
- **macOS** : pilote stub uniquement (Network Extensions prévu plus tard).
