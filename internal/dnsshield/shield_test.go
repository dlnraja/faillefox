package dnsshield

import (
	"testing"

	"github.com/dlnraja/faillefox/internal/core"
)

// TestCanonicalDomain vérifie la normalisation des noms DNS.
func TestCanonicalDomain(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"ADS.example.COM.", "ads.example.com"},
		{"clean.org", "clean.org"},
		{"UPPER.CASE", "upper.case"},
		{"x", "x"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := canonicalDomain(tc.in); got != tc.want {
			t.Errorf("canonicalDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestShieldSetBlocklist vérifie que la blocklist est bien prise en compte
// par le résolveur.
func TestShieldSetBlocklist(t *testing.T) {
	s := New(0) // port 0 : on ne bind pas, juste tester la logique.
	bl := core.NewBlocklist()
	bl.Add("evil-ads.com")
	s.SetBlocklist(bl)

	// On ne peut pas tester handleDNS sans serveur réel, mais on vérifie
	// que Contains fonctionne via la blocklist attachée.
	if !bl.Contains("tracker.evil-ads.com") {
		t.Error("sous-domaine bloqué devrait matcher")
	}
	if bl.Contains("clean.org") {
		t.Error("domaine propre ne devrait pas matcher")
	}
}

// TestShieldHasUpstreams vérifie que des upstreams sont configurés.
func TestShieldHasUpstreams(t *testing.T) {
	s := New(0)
	if len(s.upstreams) == 0 {
		t.Error("le résolveur devrait avoir au moins un upstream")
	}
	// Cloudflare et Quad9 doivent être présents (respect de la vie privée).
	found11, found99 := false, false
	for _, u := range s.upstreams {
		if len(u) >= 7 && u[:7] == "1.1.1.1" {
			found11 = true
		}
		if len(u) >= 7 && u[:7] == "9.9.9.9" {
			found99 = true
		}
	}
	if !found11 {
		t.Error("Cloudflare (1.1.1.1) devrait être un upstream")
	}
	if !found99 {
		t.Error("Quad9 (9.9.9.9) devrait être un upstream")
	}
}
