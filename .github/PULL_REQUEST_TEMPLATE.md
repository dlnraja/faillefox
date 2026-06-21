<!-- Merci de contribuer à Faillefox ! Quelques vérifications avant l'envoi. -->

## Description

<!-- Que fait cette PR ? Quel problème résout-elle ? Référence(s) d'issue : -->

Fixes #

## Type de changement

- [ ] 🐛 Correction de bug (changement ne cassant rien)
- [ ] ✨ Nouvelle fonctionnalité
- [ ] 💥 Changement cassant (breaking change)
- [ ] 📚 Documentation
- [ ] 🔧 CI / build / refactoring
- [ ] 🛡️ Sécurité

## Checklist

- [ ] `go build ./...` réussit sans erreur
- [ ] `go vet ./...` est propre
- [ ] `go test ./...` passe (tests existants + nouveaux si pertinent)
- [ ] Le code respecte le style du projet (godoc en français, fichiers focalisés)
- [ ] Je n'ai **pas** cassé la règle « l'API écoute sur loopback uniquement »
- [ ] La documentation (README, ROADMAP, docs/*) est mise à jour si nécessaire
- [ ] Je n'ai introduit **aucune** télémétrie ni appel réseau sortant du démon

## Sécurité

- [ ] Cette PR ne touche **pas** au canal de contrôle (loopback only)
- [ ] Si gestion de privilèges (admin/root) : commentée et justifiée
- [ ] Aucune information personnelle ajoutée dans les logs

## Notes pour les relecteurs

<!-- Points d'attention, choix d'implémentation à valider, etc. -->
