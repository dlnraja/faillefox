// Package auth gère l'authentification par token de l'API Faillefox.
//
// PROBLÈME DE SÉCURITÉ adressé :
//   Même en loopback, l'API était accessible sans authentification. Un
//   malware local (stealer, rançongiciel) pouvait appeler
//   POST /api/settings pour tout désactiver, ou POST /api/rules pour
//   autoriser son trafic sortant.
//
// SOLUTION :
//   Au démarrage, Faillefox génère un token aléatoire 256-bit (crypto/rand)
//   stocké dans ~/.faillefox/token (permissions 0600). Toutes les requêtes
//   API doivent le présenter en header Authorization: Bearer <token>.
//
//   L'UI web le récupère via l'URL (http://127.0.0.1:8443/?token=xxx) au
//   premier lancement, puis le stocke en sessionStorage.
//
//   Les requêtes sans token valide reçoivent un 401. Les endpoints de
//   santé (/api/status) restent accessibles sans token (pour les checks
//   de monitoring), mais toute MUTATION (POST/DELETE) exige le token.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Token est le secret d'authentification de la session courante.
type Token struct {
	value string
}

// LoadOrCreate charge le token depuis path, ou en génère un nouveau si
// absent. Le fichier est créé avec permissions 0600 (propriétaire seul).
func LoadOrCreate(path string) (*Token, error) {
	data, err := os.ReadFile(path)
	if err == nil && len(data) >= 32 {
		// Token existant valide.
		return &Token{value: strings.TrimSpace(string(data))}, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	// Génération d'un nouveau token 256-bit.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, errors.New("génération du token échouée (crypto/rand)")
	}
	token := hex.EncodeToString(buf)
	// Écriture avec permissions restrictives.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return nil, err
	}
	return &Token{value: token}, nil
}

// Value renvoie la valeur du token (pour l'afficher dans la console au
// démarrage et pour l'UI).
func (t *Token) Value() string { return t.value }

// Validate vérifie qu'un token fourni correspond au token attendu.
// Utilise subtle.ConstantTimeCompare pour éviter les attaques timing.
func (t *Token) Validate(provided string) bool {
	if t == nil || t.value == "" || provided == "" {
		return false
	}
	// subtle.ConstantTimeCompare exige des slices de même longueur.
	a := []byte(t.value)
	b := []byte(provided)
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// Middleware applique l'authentification par token. Les requêtes GET sur
// des endpoints en lecture seule (readOnlyPaths) passent sans token (pour
// le monitoring). Tout le reste exige le token.
//
// Usage :
//   mux.Use(auth.Middleware(token, readOnlyPaths))
func Middleware(token *Token, readOnlyPaths map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Les GET sur endpoints en lecture seule = pas de token requis.
			if r.Method == http.MethodGet && readOnlyPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			// Les fichiers statiques de l'UI (GET /) = pas de token
			// (l'UI récupère le token via paramètre URL puis l'envoie en header).
			if r.Method == http.MethodGet && (r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/style.css") || strings.HasPrefix(r.URL.Path, "/app.js")) {
				next.ServeHTTP(w, r)
				return
			}
			// Extraction du token : header Authorization ou query param ?token=.
			provided := r.Header.Get("Authorization")
			if strings.HasPrefix(provided, "Bearer ") {
				provided = strings.TrimPrefix(provided, "Bearer ")
			} else if q := r.URL.Query().Get("token"); q != "" {
				provided = q
			}
			if !token.Validate(provided) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "Non autorisé : token requis", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
