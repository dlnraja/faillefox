package core

import "sync"

// Profile représente un profil réseau (Maison, Bureau, Public...).
// Le profil actif détermine la politique par défaut et peut activer/désactiver
// des groupes de règles.
type Profile string

const (
	ProfileHome    Profile = "home"    // Réseau de confiance (domicile)
	ProfileOffice  Profile = "office"  // Réseau d'entreprise
	ProfilePublic  Profile = "public"  // Réseau public (café, aéroport) — le plus strict
)

// ProfileManager gère le profil réseau courant.
type ProfileManager struct {
	mu        sync.RWMutex
	active    Profile
	listeners []func(Profile)
}

// NewProfileManager crée un gestionnaire avec un profil initial.
func NewProfileManager(initial Profile) *ProfileManager {
	if initial == "" {
		initial = ProfileHome
	}
	return &ProfileManager{active: initial}
}

// Active renvoie le profil courant.
func (p *ProfileManager) Active() Profile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.active
}

// Set change le profil actif et notifie les auditeurs.
// Renvoie l'ancien profil.
func (p *ProfileManager) Set(next Profile) Profile {
	p.mu.Lock()
	old := p.active
	p.active = next
	ln := p.listeners
	p.mu.Unlock()
	if old != next {
		for _, fn := range ln {
			fn(next)
		}
	}
	return old
}

// OnChange enregistre un callback appelé à chaque changement de profil.
func (p *ProfileManager) OnChange(fn func(Profile)) {
	p.mu.Lock()
	p.listeners = append(p.listeners, fn)
	p.mu.Unlock()
}

// DefaultForProfile renvoie la politique par défaut conseillée pour un profil.
// Sur un réseau public, on bloque par défaut ; ailleurs on demande.
func DefaultForProfile(p Profile) Decision {
	switch p {
	case ProfilePublic:
		return DecisionDeny
	default:
		return DecisionAsk
	}
}
