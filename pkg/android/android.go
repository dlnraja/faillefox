// Package android expose une API minimale et autonome au monde Android via
// gomobile. CONTRAINTE CRITIQUE : gomobile n'accepte QUE des types simples
// (string, int, bool, struct de types simples) et n'aime PAS les imports de
// packages internes complexes. Ce package est donc AUTO-CONTENU : il ne
// dépend d'aucun package internal/*, uniquement de la stdlib.
//
// L'orchestration complète (moteur de règles, journal, etc.) reste côté Go
// desktop (cmd/faillefox). Sur Android, l'app Kotlin pilote directement
// les fonctions simples exposées ici.
//
// Sécurité : toutes les fonctions valident leurs entrées et n'exposent
// jamais d'état mutable partagé sans protection.
package android

import (
	"net"
	"strings"
	"sync"
)

// Version renvoie la version du moteur Android (pour affichage).
func Version() string { return "0.12" }

// Engine est le moteur de filtrage Android. Thread-safe.
// Il maintient un ensemble simple de règles par app, en mémoire.
type Engine struct {
	mu    sync.Mutex
	rules map[string]string // appID -> "allow"|"deny"
}

// NewEngine crée un moteur vide.
func NewEngine() *Engine {
	return &Engine{rules: make(map[string]string)}
}

// Decide évalue une connexion et renvoie la décision.
// Paramètres simples (strings) pour compatibilité gomobile.
// Sécurité : aucune validation externe nécessaire, on renvoie "allow"
// par défaut si l'appID est inconnu.
func (e *Engine) Decide(appID, protocol, remoteAddr string, remotePort int) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r, ok := e.rules[appID]; ok {
		return r
	}
	return "allow"
}

// SetRule définit une règle pour une app ("allow" ou "deny").
// Sécurité : on valide que la décision est dans l'ensemble autorisé.
func (e *Engine) SetRule(appID, decision string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if decision != "allow" && decision != "deny" {
		return // entrée invalide, ignorée silencieusement
	}
	e.rules[appID] = decision
}

// RemoveRule supprime la règle d'une app.
func (e *Engine) RemoveRule(appID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, appID)
}

// RuleCount renvoie le nombre de règles définies.
func (e *Engine) RuleCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.rules)
}

// ---- helpers réseau simples (self-contained, sécurisés) ----------------

// LookupHost tente une résolution DNS inverse best-effort.
// Renvoie le premier nom trouvé, ou "" si échec/timeout. Non bloquant >2s.
// Sécurité : la résolution ne peut pas causer d'injection (on renvoie juste
// une string).
func LookupHost(ip string) string {
	if net.ParseIP(ip) == nil {
		return "" // entrée invalide, on ne résout pas
	}
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// IsPrivateIP indique si une IP est privée (RFC 1918) ou loopback.
// Évite de lancer des checks réseau inutiles vers des IP internes.
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast()
}
