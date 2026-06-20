# Faillefox — Document de conception

Statut : **v0.1 (cœur + UI)** implémentée. Ce document décrit l'architecture
cible et les choix techniques validés.

## 1. Objectif

Construire un pare-feu personnel **réel**, libre et multiplateforme
(Windows / Android / Linux), piloté par application, avec une UI unifiée —
en réaction à la parodie `faillefox.com` (elle-même issue d'une perle de
CNews, 16/06/2026 — voir le README).

## 2. Décisions de conception

| Décision | Choix | Raison |
|----------|-------|--------|
| Stratégie multiplateforme | Dès le départ, cœur commun + backends natifs | Partager toute la logique métier |
| Langage du cœur | **Go** | Compilé, mémoire gérée, FFI facile, cross-compile ; cf. OpenSnitch |
| Périmètre v1 | Filtrage par application | Le plus utile aujourd'hui (cf. NetGuard/simplewall) |
| UI | Web locale (vanilla JS) embarquée | Une seule UI pour tous les OS, servie par le démon |
| Canal de contrôle | REST + SSE sur loopback uniquement | Sécurité : jamais exposé sur le réseau |
| Persistance | JSON lisible (`policies.json`) | Simple, éditable, atomique ; SQLite en v1.0 si besoin |

## 3. Architecture

Trois couches strictement séparées :

1. **`internal/core`** — le cœur. Connaît les règles, le journal, les décisions.
   **Ne sait rien** du filtrage bas niveau. Parle aux backends via une
   interface `Driver`.
2. **`internal/api`** — serveur HTTP + UI web, loopback uniquement. Pont
   entre le cœur et l'utilisateur.
3. **`internal/drivers/*`** — les backends natifs. Implémentent `Driver`.
   Un seul est actif à la fois selon la plateforme.

Cette séparation garantit que 90 % du code est partagé entre OS ; seul le
glue natif (~10 %) est spécifique.

## 4. Interface `Driver`

Contrat que chaque backend doit implémenter (cf. `internal/core/driver.go`) :

```go
type Driver interface {
    Name() string
    Start(ctx context.Context, engine *core.Engine) error
    ListApps() ([]App, error)
    ApplyRules(rules []Rule) error
    Stop() error
}
```

- `Start` lance l'interception. Pour chaque connexion détectée, le backend
  appelle `engine.Decide(conn)` et **applique** le verdict (laisser passer /
  dropper) via l'API native.
- `ListApps` expose les applications connues du système à l'UI.
- `ApplyRules` (re)installe les règles dans le mécanisme natif (no-op pour
  les backends qui interrogent le moteur en temps réel).

## 5. Moteur de décision

`Engine.Decide(conn)` parcourt les règles dans l'ordre, s'arrête à la
première qui matche (`Rule.Match`), et applique son action. Si aucune règle
ne matche, la **politique par défaut** s'applique (`ask` / `allow` / `deny`).

Chaque décision produit un `Event` qui est :
1. ajouté au **journal** (ring buffer borné en mémoire) ;
2. diffusé aux **abonnés SSE** (l'UI en direct).

## 6. Pilotes natifs — état et approche

### 6.1 `stub` (✅ implémenté)
Génère des connexions fictives à intervalle régulier. Permet de tester tout
le pipeline sans droits admin. **Aucune interception réelle.**

### 6.2 `linux-nftables` (v0.2)
- Mécanisme : `nftables` + `NFQUEUE` pour remonter les paquets en espace
  utilisateur.
- Association app ↔ connexion : lecture de `/proc/net/{tcp,udp}` croisée
  avec les inodes de socket de `/proc/<pid>/fd`.
- Droits : root requis (ou capabilities `CAP_NET_ADMIN`).

### 6.3 `windows-wfp` (v0.3)
- Mécanisme : **Windows Filtering Platform** (callouts en mode utilisateur
  via `fwpuclnt.dll`), comme simplewall.
- Association PID ↔ connexion : API IP Helper (`GetExtendedTcpTable`).
- Droits : administrateur ; exécution comme service Windows + UAC.

### 6.4 `android-vpn` (v0.4)
- Mécanisme : `VpnService` (Android impose le filtrage via une interface
  VPN locale — c'est l'approche de NetGuard/RethinkDNS).
- Cœur Go exposé à Kotlin via `gomobile bind`.
- Contrainte Android : notification permanente obligatoire pendant le VPN.

## 7. Sécurité

- **Loopback uniquement** pour l'API (bind `127.0.0.1`).
- **Écriture atomique** du store (tmp + rename).
- **Aucune télémétrie**, aucun appel réseau du démon.
- **Code source ouvert** et auditable (GPL-3.0).
- Honnêteté : la v0.1 ne filtre pas réellement (pilote stub). Documenté
  dans le README pour ne pas créer de faux sentiment de sécurité.

## 8. Limitations connues (v0.1)

- Pas d'interception réelle du trafic (pilote `stub`).
- Pas de résolution inverse DNS dans l'UI (prévu).
- Pas de notifications système pour le mode `ask` (prévu avec les pilotes natifs).
- Pas de build Android/iOS (v0.4).

## 9. Évolution future

- Mode « liste de blocage » (anti-trackers/publicités, façon Pi-hole local).
- Profils (Maison / Bureau / Public).
- Synchronisation optionnelle des règles entre appareils (jamais par défaut).
