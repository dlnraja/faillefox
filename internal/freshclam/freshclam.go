// Package freshclam automatise la mise à jour des signatures ClamAV.
//
// ClamAV ships normally avec l'outil `freshclam` qui télécharge les
// signatures (main.cvd, daily.cvd, etc.) depuis db.local.clamav.net. Sur
// certains systèmes (Windows notamment), il faut le lancer manuellement.
//
// Ce package invoque freshclam en ligne de commande à intervalle régulier
// (2h par défaut — ClamAV recommande de ne pas dépasser 24h entre deux
// updates, et 2h permet de rester à jour sur les malwares émergents).
package freshclam

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"time"
)

// Updater invoque freshclam à intervalle régulier.
type Updater struct {
	interval time.Duration
}

// New crée un updater avec une intervalle de 2h (recommandation ClamAV).
func New() *Updater {
	return &Updater{interval: 2 * time.Hour}
}

// SetInterval change la fréquence de mise à jour.
func (u *Updater) SetInterval(d time.Duration) {
	u.interval = d
}

// IsAvailable vérifie que le binaire freshclam est dans le PATH.
func (u *Updater) IsAvailable() bool {
	_, err := exec.LookPath("freshclam")
	return err == nil
}

// RunOnce met à jour les signatures une fois. Renvoie une erreur si
// freshclam n'est pas disponible ou échoue.
func (u *Updater) RunOnce(ctx context.Context) error {
	if !u.IsAvailable() {
		return errors.New("freshclam non disponible (installez ClamAV)")
	}
	log.Printf("[freshclam] mise à jour des signatures ClamAV...")
	cmd := exec.CommandContext(ctx, "freshclam", "--quiet")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[freshclam] erreur: %v (%s)", err, string(out))
		return err
	}
	log.Printf("[freshclam] signatures mises à jour")
	return nil
}

// Start lance la boucle : update immédiate puis toutes les `interval`.
// Bloquant ; à lancer dans une goroutine.
func (u *Updater) Start(ctx context.Context) {
	if err := u.RunOnce(ctx); err != nil {
		log.Printf("[freshclam] premier update: %v", err)
	}
	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := u.RunOnce(ctx); err != nil {
				log.Printf("[freshclam] update périodique: %v", err)
			}
		}
	}
}
