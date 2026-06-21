// Package gamification ajoute une couche de jeu (points, badges, streak,
// achievements) pour encourager la vigilance de l'utilisateur.
//
// Principe : un outil de sécurité qui n'est pas regardé est inutile. La
// gamification pousse l'utilisateur à consulter régulièrement son panneau,
// à valider les alertes, à maintenir une « streak » de jours protégés.
//
// Mécaniques :
//   - Points : actions de l'utilisateur (consultation, validation d'alerte,
//     blocage d'un IOC critique, mise à jour des listes).
//   - Badges : objectifs débloqués (premier blocage, 7 jours protégés,
//     100 IOC bloqués, etc.).
//   - Streak : nombre de jours consécutifs avec au moins une consultation
//     du panneau. Une streak longue = bonus de points.
//   - Niveau : dérivé des points (1 point = 1 XP, niveau = sqrt(XP/100)).
//
// Persistance : un simple fichier JSON (~/.faillefox/gamification.json),
// mis à jour à chaque action.
package gamification

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State est l'état de gamification persisté.
type State struct {
	mu          sync.Mutex
	Points      int       `json:"points"`
	Badges      []string  `json:"badges"`
	Streak      int       `json:"streak"`
	LastVisit   time.Time `json:"last_visit"`
	TotalVisits int       `json:"total_visits"`
	IOCsBlocked int       `json:"iocs_blocked"`
	AlertsValid int       `json:"alerts_validated"`
	path        string    `json:"-"`
}

// New charge l'état depuis path (ou crée un état vierge si absent).
func New(path string) (*State, error) {
	s := &State{path: path}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, s)
	}
	return s, nil
}

// Save persiste l'état (écriture atomique).
func (s *State) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Level calcule le niveau courant à partir des points (XP).
// Niveau = floor(sqrt(points / 100)). 100 pts = niveau 1, 400 = 2, 900 = 3...
func (s *State) Level() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int(math.Sqrt(float64(s.Points) / 100))
}

// Action signale une action utilisateur et attribue des points. Renvoie les
// éventuels nouveaux badges débloqués.
type Action string

const (
	ActionVisit         Action = "visit"          // consultation du panneau
	ActionAlertValidated Action = "alert_validated" // validation d'une alerte
	ActionIOCBlocked    Action = "ioc_blocked"    // IOC bloqué
	ActionListUpdated   Action = "list_updated"   // listes mises à jour
	ActionScanRun       Action = "scan_run"       // scan ClamAV/YARA lancé
)

// pointsFor renvoie les points attribués à une action.
func pointsFor(a Action) int {
	switch a {
	case ActionVisit:
		return 5
	case ActionAlertValidated:
		return 50
	case ActionIOCBlocked:
		return 30
	case ActionListUpdated:
		return 10
	case ActionScanRun:
		return 20
	}
	return 0
}

// Record applique une action, met à jour streak/points/badges, et renvoie
// les nouveaux badges débloqués (à afficher dans l'UI).
func (s *State) Record(a Action) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	pts := pointsFor(a)
	s.Points += pts

	switch a {
	case ActionVisit:
		s.handleVisit()
	case ActionIOCBlocked:
		s.IOCsBlocked++
	case ActionAlertValidated:
		s.AlertsValid++
	}

	// Vérification des badges.
	var newBadges []string
	candidates := s.badgeCandidates()
	for _, b := range candidates {
		if !contains(s.Badges, b) {
			s.Badges = append(s.Badges, b)
			newBadges = append(newBadges, b)
		}
	}
	return newBadges
}

// handleVisit gère la logique de streak lors d'une consultation.
func (s *State) handleVisit() {
	now := time.Now()
	s.TotalVisits++
	if s.LastVisit.IsZero() {
		s.Streak = 1
	} else {
		// Même jour : pas d'incrémentation.
		if sameDay(s.LastVisit, now) {
			// streak inchangée
		} else if daysBetween(s.LastVisit, now) == 1 {
			s.Streak++ // jour consécutif
		} else {
			s.Streak = 1 // streak cassée
		}
	}
	// Bonus de streak.
	if s.Streak > 1 {
		s.Points += s.Streak
	}
	s.LastVisit = now
}

// badgeCandidates renvoie les badges que l'utilisateur vient de débloquer
// (sans vérifier s'il les avait déjà).
func (s *State) badgeCandidates() []string {
	var out []string
	if s.TotalVisits >= 1 {
		out = append(out, "first-visit")
	}
	if s.Streak >= 7 {
		out = append(out, "streak-7")
	}
	if s.Streak >= 30 {
		out = append(out, "streak-30")
	}
	if s.IOCsBlocked >= 1 {
		out = append(out, "first-block")
	}
	if s.IOCsBlocked >= 100 {
		out = append(out, "block-100")
	}
	if s.AlertsValid >= 10 {
		out = append(out, "vigilant")
	}
	if s.Points >= 1000 {
		out = append(out, "guardian")
	}
	return out
}

// BadgeDescriptions mappe un ID de badge à sa description lisible.
var BadgeDescriptions = map[string]string{
	"first-visit": "🦊 Bienvenue — première consultation du panneau",
	"streak-7":    "🔥 7 jours consécutifs de vigilance",
	"streak-30":   "🏆 30 jours consécutifs — habitué !",
	"first-block": "🛡️ Premier IOC bloqué",
	"block-100":   "💪 100 menaces bloquées",
	"vigilant":    "👁️ 10 alertes validées",
	"guardian":    "👑 Gardien — 1000 points atteints",
}

// ---- helpers --------------------------------------------------------------

func contains(slice []string, v string) bool {
	for _, s := range slice {
		if s == v {
			return true
		}
	}
	return false
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func daysBetween(a, b time.Time) int {
	a = time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, a.Location())
	b = time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, b.Location())
	return int(b.Sub(a).Hours() / 24)
}
