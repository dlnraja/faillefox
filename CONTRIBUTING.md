# Contribuer à Faillefox

Merci de votre intérêt ! Faillefox est un projet jeune : toute aide compte,
en particulier sur les **pilotes natifs** (Linux nftables, Windows WFP,
Android VPNService) qui demandent une expertise par OS.

## Par où commencer

- Consultez la [feuille de route](../README.md#-feuille-de-route) dans le README.
- Les issues marquées `good first issue` sont des points d'entrée idéaux.
- La v0.2 prioritaire est le **pilote Linux nftables** (voir
  `docs/design.md` §6.2).

## Pré-requis

- [Go](https://go.dev/dl/) 1.26+
- Pour travailler sur l'UI : un navigateur moderne suffit (vanilla JS, aucun
  build).
- Pour les pilotes natifs : les SDK/headers correspondants
  (Windows SDK, Android NDK, etc.).

## Lancer le projet en développement

```bash
git clone https://github.com/dlnraja/faillefox.git
cd faillefox
go run ./cmd/faillefox -port 8443 -data ./_testdata
```

Puis ouvrez http://127.0.0.1:8443. Le pilote `stub` simule des connexions
toutes les 3 secondes : vous verrez le journal se remplir en direct.

## Avant de soumettre une PR

1. `go build ./...` doit réussir sans erreur.
2. `go vet ./...` doit être propre.
3. Ajoutez des tests pour toute nouvelle logique dans `internal/core`.
4. Respectez le style existant : godoc en français, packages focalisés,
   fichiers courts.
5. Ne cassez **jamais** la règle « l'API écoute sur loopback uniquement ».

## Ajouter un nouveau pilote natif

1. Créez un package `internal/drivers/<nom>`.
2. Implémentez `core.Driver`.
3. Dans un `init()`, appelez `core.RegisterDriver("<nom>", New)`.
4. Documentez le mécanisme dans `docs/design.md`.
5. Ajoutez une ligne au tableau de comparaison du README si pertinent.

## Sécurité

Faillefox est un outil de sécurité : la rigueur est non négociable.

- **Pas de télémétrie**, pas d'appel réseau initié par le démon.
- **Loopback uniquement** pour le canal de contrôle.
- Toute gestion de privilèges (admin/root) doit être revue avec attention.
- Signalez en privé toute vulnérabilité avant d'ouvrir une issue publique.

## Code de conduite

Soyez respectueux et constructif. Ce projet n'a aucune tolérance pour les
comportements toxiques, envers quiconque.
