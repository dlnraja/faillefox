// Package middleware fournit les middlewares de sécurité HTTP du serveur
// API de Faillefox. Ils sont appliqués à TOUS les endpoints.
//
// Sécurité non négociable pour un pare-feu :
//   - CSP stricte (pas de scripts inline, pas de sources externes)
//   - Headers de durcissement (X-Frame-Options, X-Content-Type-Options...)
//   - Rate limiting (anti-brute-force sur l'API loopback)
//   - Validation de l'origine (uniquement loopback)
package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityHeaders ajoute les headers HTTP de sécurité standard. Comme le
// panneau est servi en loopback sans scripts externes, on peut être très
// strict sur la CSP.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// CSP stricte : pas de scripts externes, pas d'inline (sauf ce qu'on
		// maîtrise), pas de frames, pas de plugins.
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "1; mode=block")
		// Pas de HSTS en loopback (HTTP, pas HTTPS) — serait contre-productif.
		next.ServeHTTP(w, r)
	})
}

// LoopbackOnly refuse les connexions venant d'une IP non-loopback.
// C'est une défense en profondeur : même si le bind tombe sur 0.0.0.0 par
// erreur de config, les requêtes externes sont rejetées.
func LoopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil || (!ip.IsLoopback()) {
			http.Error(w, "Accès refusé : loopback uniquement", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimiter limite le nombre de requêtes par IP et par fenêtre temporelle.
// Protège l'API loopback contre un éventuel abus (process local défaillant,
// ou un malware local qui tenterait de brute-forcer les endpoints).
type RateLimiter struct {
	mu      sync.Mutex
	visits map[string][]time.Time // IP -> timestamps
	max     int                   // max requêtes par fenêtre
	window  time.Duration
}

// NewRateLimiter crée un limiteur : max requêtes par window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visits: make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

// Middleware renvoie le handler HTTP qui applique le rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		if rl.isRateLimited(host) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Trop de requêtes", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isRateLimited renvoie true si l'IP a dépassé le quota. Thread-safe.
func (rl *RateLimiter) isRateLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	// Filtre les anciens timestamps.
	recent := rl.visits[ip][:0]
	for _, t := range rl.visits[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rl.max {
		rl.visits[ip] = recent
		return true
	}
	rl.visits[ip] = append(recent, now)
	return false
}

// SanitizePath nettoie un chemin fourni par l'utilisateur pour prévenir le
// path traversal (../../etc/passwd). Renvoie le chemin nettoyé ou "" si
// suspect.
func SanitizePath(input string) string {
	if input == "" {
		return ""
	}
	// Refuse les tentatives de remontée de répertoire.
	if strings.Contains(input, "..") {
		return ""
	}
	// Nettoie les séparateurs mixtes.
	cleaned := strings.ReplaceAll(input, "\x00", "")
	return cleaned
}

// ValidateDecision vérifie qu'une décision de règle est dans l'ensemble
// autorisé. Évite qu'une valeur arbitraire soit stockée puis interprétée.
func ValidateDecision(d string) bool {
	switch d {
	case "allow", "deny", "ask":
		return true
	}
	return false
}
