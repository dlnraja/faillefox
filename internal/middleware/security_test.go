package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSanitizePathRejectsTraversal vérifie le blocage du path traversal.
func TestSanitizePathRejectsTraversal(t *testing.T) {
	cases := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"/etc/../passwd",
	}
	for _, c := range cases {
		if SanitizePath(c) != "" {
			t.Errorf("path traversal %q devrait être rejeté", c)
		}
	}
}

// TestSanitizePathAcceptsValid vérifie qu'un chemin normal passe.
func TestSanitizePathAcceptsValid(t *testing.T) {
	valid := "/tmp/Documents/rapport.pdf"
	if got := SanitizePath(valid); got != valid {
		t.Errorf("chemin valide rejeté: %q -> %q", valid, got)
	}
}

// TestSanitizePathStripsNullBytes vérifie l'injection de null bytes.
func TestSanitizePathStripsNullBytes(t *testing.T) {
	input := "file\x00.txt"
	got := SanitizePath(input)
	if strings.Contains(got, "\x00") {
		t.Errorf("null byte devrait être supprimé: %q", got)
	}
}

// TestValidateDecision vérifie la validation des décisions.
func TestValidateDecision(t *testing.T) {
	valid := []string{"allow", "deny", "ask"}
	for _, d := range valid {
		if !ValidateDecision(d) {
			t.Errorf("%q devrait être valide", d)
		}
	}
	invalid := []string{"", "block", "ACCEPT", "rm -rf", "'; DROP TABLE"}
	for _, d := range invalid {
		if ValidateDecision(d) {
			t.Errorf("%q ne devrait pas être valide", d)
		}
	}
}

// TestRateLimiterBlocksAfterMax vérifie le blocage après dépassement.
func TestRateLimiterBlocksAfterMax(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if rl.isRateLimited("1.2.3.4") {
			t.Errorf("requête %d ne devrait pas être limitée", i+1)
		}
	}
	if !rl.isRateLimited("1.2.3.4") {
		t.Error("la 4e requête devrait être limitée (max=3)")
	}
}

// TestRateLimiterDifferentIPs vérifie que les IP sont indépendantes.
func TestRateLimiterDifferentIPs(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	if rl.isRateLimited("1.1.1.1") {
		t.Error("1ère IP ne devrait pas être limitée")
	}
	if rl.isRateLimited("1.1.1.1") {
		// OK, 2e fois même IP = limitée
	}
	if rl.isRateLimited("2.2.2.2") {
		t.Error("2e IP différente ne devrait pas être limitée")
	}
}

// TestSecurityHeaders vérifie que les headers sont bien posés.
func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("CSP header manquant")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options manquant")
	}
}

// TestLoopbackOnlyRejectsExternal vérifie le rejet des IP externes.
func TestLoopbackOnlyRejectsExternal(t *testing.T) {
	handler := LoopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	// IP externe simulée.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("IP externe devrait être 403, got %d", rec.Code)
	}
	// IP loopback devrait passer.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("IP loopback devrait être 200, got %d", rec2.Code)
	}
}
