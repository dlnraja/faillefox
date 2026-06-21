# Threat intelligence & corrélation

Faillefox intègre une couche de **threat intelligence** qui agrège
automatiquement les indicateurs de compromission (IOC) publics et les
**croise** pour prioriser les alertes.

## Sources intégrées (toutes gratuites, publiques)

| Source | Type d'IOC | URL | Sans clé API |
|--------|-----------|-----|--------------|
| **Abuse.ch ThreatFox** | IP/domaines/hashes attribués à APT | https://threatfox.abuse.ch | ✅ |
| **Abuse.ch URLhaus** | URLs malveillantes + hashes | https://urlhaus.abuse.ch | ✅ |
| **AlienVault OTX** | Pulses communautaires (IOC variés) | https://otx.alienvault.com | ✅ (quota réduit) |
| **StevenBlack/hosts** | Domaines malveillants (déjà dans updater) | https://github.com/StevenBlack/hosts | ✅ |

### Ajouter MISP (optionnel)

MISP est une plateforme open source de partage de threat intel. Pour brancher
une instance MISP :
1. Déployez votre propre MISP (https://www.misp-project.org/) — gratuit.
2. Ajoutez un feed MISP dans `internal/threatintel/aggregator.go` (fonction
   `sources()`) — l'API MISP est publique et documentée.

## Corrélateur d'alertes

Le corrélateur (`internal/correlate`) croise plusieurs signaux pour produire
des alertes **priorisées** plutôt qu'une liste brute :

| Signal | Points |
|--------|--------|
| IOC vu par 1 source threat intel | +30 |
| IOC vu par 2 sources | +60 |
| IOC vu par 3 sources | +90 |
| Logiciel émetteur a une CVE connue | +40 |
| Réseau public (profil) | +20 |
| Bonus de streak (gamification) | +N |

### Niveaux de sévérité

| Score | Sévérité | Couleur UI |
|-------|----------|------------|
| < 30 | (pas d'alerte) | — |
| 30-59 | LOW | bleu |
| 60-89 | MEDIUM | orange |
| 90-119 | HIGH | rouge clair |
| ≥ 120 | CRITICAL | rouge vif |

Exemple concret : une connexion vers une IP présente dans Abuse.ch **et** OTX
**et** émise par un logiciel avec CVE ouverte sur un réseau public →
score = 90 + 40 + 20 = **150 (CRITICAL)**.

## Scanner YARA (règles publiques)

Faillefox peut charger des règles YARA **publiques** (YARA Forge,
signature-base, règles communautaires) et scanner des fichiers à la
recherche de patterns connus.

```bash
./faillefox -yara-rules ./rules.yar
```

### ⚠️ Honnêteté sur le moteur YARA

- Le package `yarascan` n'embarque **pas** le moteur YARA complet (qui
  nécessiterait `libyara` en C + CGO, lourd et non portable).
- À la place, il implémente un chargeur de règles YARA simplifié qui extrait
  les patterns de chaînes (`$s = "..."`) et les cherche dans les fichiers.
- Cela couvre ~80 % des règles courantes mais **pas** les conditions
  complexes, les modules PE/ELF, ni les caractères génériques hex.
- Pour le moteur complet : intégrer `github.com/hillu/go-yara` (CGO) — v0.8.

### Sources de règles YARA publiques recommandées

- **YARA Forge** : https://github.com/InQuest/yara-rules
- **signature-base** (Neo23x0) : https://github.com/Neo23x0/signature-base
- **YARA rules community** : https://github.com/Yara-Rules/rules

### On NE génère PAS de règles maison

Écrire des signatures AV maison sans infrastructure de test (sandboxing,
reverse engineering, validation sur corpus de malwares + de logiciels légitimes)
produit des **faux positifs massifs** (qui peuvent casser le système de
l'utilisateur) ou des **faux négatifs** (faux sentiment de sécurité).
Faillefox ne fait que **charger** des règles éprouvées écrites par des
analystes professionnels.

## Gamification

La gamification (`internal/gamification`) encourage la consultation
régulière du panneau :

| Action | Points |
|--------|--------|
| Consultation du panneau | +5 |
| IOC bloqué | +30 |
| Alerte validée | +50 |
| Scan lancé | +20 |
| Listes mises à jour | +10 |
| Bonus de streak | +N (N = jours consécutifs) |

### Badges débloquables

| Badge | Condition |
|-------|-----------|
| 🦊 `first-visit` | Première consultation |
| 🔥 `streak-7` | 7 jours consécutifs |
| 🏆 `streak-30` | 30 jours consécutifs |
| 🛡️ `first-block` | Premier IOC bloqué |
| 💪 `block-100` | 100 menaces bloquées |
| 👁️ `vigilant` | 10 alertes validées |
| 👑 `guardian` | 1000 points atteints |

### Niveau

Niveau = floor(sqrt(points / 100)). 100 pts = niveau 1, 400 = 2, 900 = 3...

La gamification est **activée par défaut** (désactivable via `-gamification=false`).
Les données sont persistées dans `~/.faillefox/gamification.json`.

## Activation

```bash
# Tout activer (threat intel + YARA + gamification)
./faillefox -threat-intel -yara-rules ./rules.yar -gamification

# Gamification seule (par défaut)
./faillefox
```
