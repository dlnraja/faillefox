# Faillefox — Présentation du projet

> Un VRAI pare-feu, gratuit, libre, multiplateforme. Né d'une blague.
> L'anti-faillefox.com.

---

## Le pitch en 30 secondes

**Faillefox** est un pare-feu personnel open source (GPL-3.0) qui fonctionne
sur **Windows, Android et Linux** avec une interface unique. Il laisse
l'utilisateur décider, application par application, ce qui a le droit de
sortir sur Internet.

Son nom est un clin d'œil : il naît en **réaction** à la parodie
[faillefox.com](https://faillefox.com), un faux pare-feu au slogan
« *Faillefox fait tout. Sauf vous protéger.* ». Notre version, elle,
protège réellement.

## L'origine de l'histoire

- **16 juin 2026** : sur CNews, l'essayiste Joseph Macé-Scaron, invité pour
  parler d'IA, s'embrouille en direct et prononce :
  > *« Fox, on a détecté de près de 300 failles dans Fox […] C'est le parfeu.
  > C'est le parfeu pour tout… pardonnez-moi. »**
  
  ([Source — YouTube CNews, 16/06/2026](https://www.youtube.com/watch?v=aZZGPZ4l0_Q))
- À la suite de cette séquence, le site
  **[faillefox.com](https://faillefox.com)** apparaît : une parodie de
  pare-feu qui « fait tout sauf protéger », affichant « 461 failles incluses ».
- **Faillefox (ce dépôt)** prend l'idée à contre-pied : si un « pare-feu pour
  tout » existe en blague, pourquoi n'existerait-il pas **pour de vrai**,
  en libre, multiplateforme et efficace ?

> Ce projet n'est pas affilié à CNews, à M. Macé-Scaron ni au site parodique
> faillefox.com. Les propos cités sont publics et reconnus par leur auteur.

## Ce qui rend Faillefox différent

| | Pare-feu classique | Faillefox |
|---|---|---|
| Plateformes | Un seul OS généralement | **Windows + Android + Linux** |
| UI | Par OS | **Une seule UI web locale** |
| Cœur | C/C++ propriétaire | **Go, open source, auditable** |
| Transparence | Boîte noire | **Code source, sans télémétrie** |
| Modèle économique | Freemium / pubs | **Gratuit, libre (GPL-3.0)** |

## Ce qu'on peut faire aujourd'hui (v0.2)

- ✅ **Filtrage par application** sur Windows (via Pare-feu Windows) et
  Linux (nftables/iptables).
- ✅ **Mode simple** (interrupteurs par app) + **mode avancé** (règles
  port/IP/protocole).
- ✅ **Journal temps réel** des connexions interceptées (SSE).
- ✅ **Blocklist anti-trackers/publicités** optionnelle (façon Pi-hole local).
- ✅ **Profils réseau** (Maison / Bureau / Public).
- ✅ **Journal persistant rotatif** (JSONL sur disque).
- ✅ **Panneau web local** sur `127.0.0.1`, jamais exposé au réseau.
- ✅ **Scaffold Android** complet (VpnService + Kotlin + gomobile).

## Feuille de route (extrait)

- v0.3 — Pilote WFP avancé (filtrage par app strict, prompt temps réel)
- v0.4 — Android : forward des paquets (tun2socks), UI détaillée
- v1.0 — Stabilisation, signature automatique, documentation grand public

Voir [`ROADMAP.md`](ROADMAP.md) pour le détail.

## Pour les curieux / contributeurs

- [README.md](../README.md) — présentation complète + démarrage rapide
- [docs/design.md](design.md) — architecture technique
- [docs/antivirus.md](antivirus.md) — signature & faux positifs AV
- [docs/android.md](android.md) — build de l'app Android
- [CONTRIBUTING.md](../CONTRIBUTING.md) — contribuer au projet

## Licence

GNU GPL v3.0 — parce qu'un outil de sécurité doit rester libre et auditable.
