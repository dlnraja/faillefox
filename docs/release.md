# Processus de release

Comment Faillefox passe d'un commit à une release publiée — et pourquoi
c'est presque entièrement automatique.

## Vue d'ensemble du pipeline

```
                ┌─────────────────────────────────────────────┐
                │  développeur pousse un commit sur master     │
                └────────────────────┬────────────────────────┘
                                     │
                    ┌────────────────▼─────────────────┐
                    │  ci.yml + lint.yml               │
                    │  build + vet + test + lint        │
                    │  (matrice Ubuntu/Win/macOS)       │
                    └────────────────┬─────────────────┘
                                     │ (sur master uniquement)
                    ┌────────────────▼─────────────────┐
                    │  auto-version.yml                 │
                    │  1. analyse les conventional       │
                    │     commits depuis le dernier tag  │
                    │  2. détermine le bump (major/minor/│
                    │     patch) — ou skip si docs/chore │
                    │  3. met à jour version_info.json   │
                    │  4. ajoute une entrée CHANGELOG.md │
                    │  5. commit + push [skip release]   │
                    │  6. crée & pousse le tag vX.Y.Z    │
                    └────────────────┬─────────────────┘
                                     │ (push de tag)
                    ┌────────────────▼─────────────────┐
                    │  release.yml                      │
                    │  - goversioninfo (VERSIONINFO Win)│
                    │  - cross-compile 6 binaires        │
                    │  - archives zip/tar.gz + SHA256SUMS│
                    │  - GitHub Release (notes auto)     │
                    └────────────────┬─────────────────┘
                                     │ (release publiée)
                    ┌────────────────▼─────────────────┐
                    │  sign.yml (si SignPath configuré)  │
                    │  - signature Authenticode du .exe  │
                    │  - re-upload du .exe signé         │
                    └───────────────────────────────────┘
```

## Conventional commits (obligatoires pour l'auto-version)

L'auto-versionning se base sur le **préfixe** des messages de commit :

| Préfixe | Effet | Exemple |
|---------|-------|---------|
| `feat:` | bump **MINOR** (0.5.0 → 0.6.0) | `feat(api): endpoint /api/scan` |
| `fix:` | bump **PATCH** (0.5.0 → 0.5.1) | `fix(dns): crash sur NXDOMAIN` |
| `feat!:` / `fix!:` | bump **MAJOR** (0.5.0 → 1.0.0) | `feat!: refonte API` |
| `docs:` / `chore:` / `ci:` | **pas de release** | `docs: typo README` |
| `[skip release]` dans le msg | **pas de release** | `chore: merge PR [skip release]` |

Exemples valides :
```
feat(ui): onglet Scan ClamAV dans le tableau de bord
fix(clamscan): parsing sortie clamd avec chemins UNC
docs: guide de compilation multi-OS
```

## Comment faire une release

### Cas 1 : release automatique (par défaut)

1. Fusionnez votre PR vers `master` avec un message conventional commit.
2. L'auto-version fait tout : bump, CHANGELOG, tag, release.
3. La release apparaît sur https://github.com/dlnraja/faillefox/releases
   en ~3 minutes.

### Cas 2 : release manuelle (tag explicite)

```bash
git tag -a v0.6.0 -m "Faillefox v0.6.0"
git push origin v0.6.0
```

Le workflow `release.yml` se déclenche directement (cross-compile + release).

### Cas 3 : pas de release (hotfix docs uniquement)

Ajoutez `[skip release]` au message de commit, ou utilisez un préfixe
`docs:` / `chore:` / `ci:`.

## Contenu d'une release

Chaque release contient :

| Asset | Description |
|-------|-------------|
| `faillefox-windows-amd64.exe.zip` | Windows x64 + LICENSE + README + ANTIVIRUS.txt |
| `faillefox-windows-arm64.exe.zip` | Windows ARM64 |
| `faillefox-linux-amd64.tar.gz` | Linux x64 |
| `faillefox-linux-arm64.tar.gz` | Linux ARM64 (Raspberry Pi) |
| `faillefox-darwin-amd64.tar.gz` | macOS Intel |
| `faillefox-darwin-arm64.tar.gz` | macOS Apple Silicon |
| `SHA256SUMS` | Sommes de contrôle de toutes les archives |

Les notes de release sont **générées automatiquement** depuis les PRs/commits
depuis le tag précédent.

## Auto-release hebdomadaire

En plus de l'auto-version sur commit, un workflow `weekly-release.yml` tourne
chaque lundi à 04:00 UTC. S'il y a des commits non publiés depuis le dernier
tag, il génère un tag `v0.<année>.<semaine>.<seq>` et publie une release.

→ Voir [ROADMAP.md](../ROADMAP.md) §v0.4 pour le détail.

## Vérifier une release

Téléchargez l'archive + `SHA256SUMS`, puis :

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

Doit afficher `OK` pour chaque archive téléchargée.

## Signature des binaires Windows

Si SignPath est configuré (secrets GitHub), le workflow `sign.yml` signe
automatiquement le `.exe` Windows après chaque release. Voir
[docs/antivirus.md](antivirus.md).
