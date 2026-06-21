# ClamAV — intégration et limites honnêtes

Faillefox intègre le moteur **ClamAV** pour le scan de fichiers à la demande.
Cette page explique comment l'installer et — surtout — **ce qu'il peut et ne
peut pas faire**, pour ne pas créer de faux sentiment de sécurité.

---

## ⚠️ ClamAV vs Kaspersky / Bitdefender : la vérité

ClamAV est **le seul moteur antivirus open source** largement déployé et
maintenu (par Cisco Talos). Mais il est **nettement inférieur** aux solutions
commerciales sur plusieurs points :

| Critère | ClamAV | Kaspersky / Bitdefender / ESET |
|---------|--------|--------------------------------|
| Détection par signatures | ✅ Bonne | ✅ Excellente |
| Heuristique / ML | ⚠️ Basique | ✅ Avancée (apprentissage) |
| Détection comportementale | ❌ Faible | ✅ Temps réel |
| Sandbox (analyse bac à sable) | ❌ Non | ✅ Intégrée |
| Protection temps réel | ⚠️ Possible (clamdscan) | ✅ Native |
| Taux de détection zero-day | ❌ Faible | ✅ Élevé |
| Coût | ✅ **Gratuit, libre** | ❌ ~30-60 €/an |
| Code auditable | ✅ Oui | ❌ Fermé |

**Conclusion honnête** : ClamAV est utile pour
- scanner un fichier téléchargé **avant** de l'exécuter ;
- vérifier une clé USB / une archive ;
- détecter des malwares **connus** dans une collection de fichiers.

**ClamAV NE REMPLACE PAS** une véritable solution antivirus temps réel.
Pour une protection complète, combinez Faillefox + ClamAV **avec** Windows
Defender (gratuit, intégré à Windows) ou un AV commercial.

---

## Installation de ClamAV

### Windows
1. Téléchargez l'installateur officiel : https://www.clamav.net/downloads
2. Installez (le daemon `clamd` est optionnel mais recommandé).
3. Démarrez le service ClamAV (Services Windows).

### Linux (Debian/Ubuntu)
```bash
sudo apt install clamav clamav-daemon
sudo systemctl enable --now clamav-daemon
sudo freshclam   # télécharge les signatures
```

### macOS
```bash
brew install clamav
brew services start clamav
```

---

## Utilisation dans Faillefox

Activez le scanner ClamAV au démarrage :

```bash
./faillefox -clamav
```

Faillefox détecte automatiquement :
- le **daemon clamd** (port 3310) → scans rapides, idéaux en masse ;
- sinon, le **binaire clamscan** → scan ponctuel en ligne de commande.

Au démarrage, Faillefox log l'état de ClamAV :
- `clamd détecté` → mode daemon (rapide)
- `clamscan détecté` → mode CLI (plus lent)
- `NON disponible` → ClamAV n'est pas installé, scan désactivé

---

## Limitations actuelles (v0.3)

- Scan **à la demande** uniquement (pas de surveillance temps réel des
  exécutions). La surveillance temps réel via `clamdscan --watch` est prévue.
- Pas d'UI de scan dans le panneau web (à venir v0.4).
- Pas de mise à jour automatique des signatures (utilisez `freshclam`
  côté système, c'est standard).

## Sources
- Site officiel ClamAV : https://www.clamav.net/
- Documentation : https://docs.clamav.net/
- Code source : https://github.com/Cisco-Talos/clamav (GPL-2.0)
