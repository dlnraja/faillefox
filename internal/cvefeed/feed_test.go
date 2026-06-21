package cvefeed

import (
	"testing"
)

// TestExtractCPEField vérifie le parsing des CPE 2.3.
// Format CPE : cpe:2.3:a:vendor:product:version:...
func TestExtractCPEField(t *testing.T) {
	cpe := "cpe:2.3:a:curl:curl:7.74.0:*:*:*:*:*:*:*"
	if got := extractCPEField(cpe, "vendor"); got != "curl" {
		t.Errorf("vendor: got %q, want curl", got)
	}
	if got := extractCPEField(cpe, "product"); got != "curl" {
		t.Errorf("product: got %q, want curl", got)
	}
	if got := extractCPEField(cpe, "version"); got != "7.74.0" {
		t.Errorf("version: got %q, want 7.74.0", got)
	}
}

// TestExtractCPEFieldShort vérifie le comportement sur un CPE tronqué.
func TestExtractCPEFieldShort(t *testing.T) {
	short := "cpe:2.3:a:apache"
	if got := extractCPEField(short, "product"); got != "" {
		t.Errorf("CPE tronqué devrait renvoyer vide, got %q", got)
	}
}

// TestTruncate vérifie la troncature avec ellipse.
func TestTruncate(t *testing.T) {
	if got := truncate("court", 10); got != "court" {
		t.Errorf("chaîne courte devrait rester intacte: %q", got)
	}
	got := truncate("abcdefghijklmnopqrstuvwxyz", 10)
	// truncate renvoie 9 caractères + "…" (3 octets en UTF-8).
	if got[:9] != "abcdefghi" {
		t.Errorf("préfixe incorrect: %q", got)
	}
	if !endsWithEllipsis(got) {
		t.Errorf("devrait se terminer par l'ellipse: %q", got)
	}
}

// endsWithEllipsis helper de test (le caractère U+2026 tient sur 3 octets).
func endsWithEllipsis(s string) bool {
	return len(s) >= 3 && s[len(s)-3:] == "…"
}

// TestCheckSoftwareEmpty vérifie le comportement avec un index vide.
func TestCheckSoftwareEmpty(t *testing.T) {
	f := New() // index vide, pas de RefreshAll
	alerts := f.CheckSoftware([]Software{{Name: "curl", Version: "7.74.0"}})
	if len(alerts) != 0 {
		t.Errorf("index vide ne doit produire aucune alerte, got %d", len(alerts))
	}
}

// TestCheckSoftwareMatch vérifie la correspondance par nom de produit.
// On peuple manuellement l'index pour tester la logique sans appel réseau.
func TestCheckSoftwareMatch(t *testing.T) {
	f := New()
	f.idx["curl"] = []cveEntry{
		{cve: "CVE-2023-38545", severity: "HIGH", description: " SOCKP5 heap overflow"},
	}
	alerts := f.CheckSoftware([]Software{
		{Name: "curl", Version: "7.74.0"},
		{Name: "bash", Version: "5.0"},
	})
	if len(alerts) != 1 {
		t.Fatalf("1 alerte attendue pour curl, got %d", len(alerts))
	}
	if alerts[0].CVE != "CVE-2023-38545" {
		t.Errorf("mauvaise CVE: %s", alerts[0].CVE)
	}
	if alerts[0].Severity != "HIGH" {
		t.Errorf("mauvaise sévérité: %s", alerts[0].Severity)
	}
}
