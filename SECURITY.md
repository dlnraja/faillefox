# Politique de sécurité

Faillefox est un outil de sécurité : nous prenons les signalements de
vulnérabilité au sérieux. Merci de nous aider à le rendre plus sûr.

## Versions supportées

Seules les versions publiées dans les [releases](https://github.com/dlnraja/faillefox/releases)
sont supportées. La branche `master` est en développement continu et n'est
pas considérée comme stable.

| Version | Supportée |
|---------|-----------|
| v0.5.x  | ✅ |
| v0.4.x  | ✅ |
| v0.3.x  | ⚠️ correctifs critiques uniquement |
| < v0.3  | ❌ |

## Comment signaler une vulnérabilité

> ⚠️ **NE PAS ouvrir une issue publique** pour une vulnérabilité exploitable.

### Procédure privée (recommandée pour les failles sensibles)

1. Utilisez les **GitHub Security Advisories** :
   https://github.com/dlnraja/faillefox/security/advisories/new
2. Décrivez : le problème, son impact, une reproduction, et si possible
   une suggestion de correctif.
3. Vous recevrez un accusé de réception sous **72 h**.
4. Nous travaillons ensemble à un correctif et coordonnons la divulgation
   (crédit à l'auteur du signalement, sauf demande contraire).

### Procédure publique (CVE de dépendances, faux positifs AV)

Pour les problèmes **non sensibles** (ex: une CVE connue affectant une
dépendance Go, un faux positif antivirus), vous pouvez ouvrir une issue
avec le [modèle de signalement de sécurité](.github/ISSUE_TEMPLATE/security.md).

## Périmètre couvert

| Composant | Couvert |
|-----------|---------|
| Cœur Go (`internal/core`) | ✅ |
| API HTTP + UI web (`internal/api`) | ✅ |
| Pilotes natifs (netfw, nftables) | ✅ |
| DNS sinkhole, CVE feed, ClamAV | ✅ |
| Configuration par défaut | ✅ |
| Infrastructure CI/CD | ✅ |
| Versions EOL / abandonnées | ❌ |

## Principes de sécurité du projet

Ces principes sont **non négociables** et toute PR les enfreignant doit
être refusée :

1. **Loopback uniquement** : le serveur HTTP bind sur `127.0.0.1`, jamais
   sur `0.0.0.0`. Le canal de contrôle ne doit jamais être joignable depuis
   le réseau.
2. **Aucune télémétrie** : aucun appel réseau sortant initié par le démon,
   sauf rafraîchissement explicite des listes publiques (DNS/CVE) demandé
   par l'utilisateur.
3. **Sûreté mémoire** : cœur en Go (GC, pas de débordements de tampon).
4. **Droits minimaux** : élévation admin/root uniquement quand strictement
   nécessaire (netsh/nftables), justifiée et documentée.
5. **Honnêteté** : on ne prétend jamais une protection qu'on n'apporte pas.
   ClamAV est documenté comme limité vs solutions commerciales.

## Mesures de sécurité déjà en place

- ✅ Canal de contrôle loopback uniquement (jamais exposé au réseau)
- ✅ Persistance en écriture atomique (tmp + rename) pour éviter la corruption
- ✅ Aucune télémétrie par défaut
- ✅ Code source ouvert (GPL-3.0), auditable
- ✅ Métadonnées VERSIONINFO + manifeste UAC `asInvoker` sur les binaires Windows
- ✅ Workflow SignPath prêt pour la signature Authenticode
- ✅ CI multi-OS avec `go vet` + `golangci-lint` sur chaque PR
- ✅ Dependabot pour les mises à jour de dépendances

## Programmes de récompense

Faillefox est un projet open source bénévole : aucune récompense financière
n'est offerte. Les rapporteurs sont **crédités** dans le CHANGELOG et
l'advisory GitHub (sauf demande d'anonymat).
