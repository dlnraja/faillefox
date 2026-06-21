package threatintel

import (
	"testing"
)

// TestAddLookup vérifie l'ajout et la recherche d'IOC.
func TestAddLookup(t *testing.T) {
	a := New()
	a.Add(IOC{Value: "1.2.3.4", Type: IOCIP, Source: "abuse.ch", Confidence: 8})
	a.Add(IOC{Value: "1.2.3.4", Type: IOCIP, Source: "otx", Confidence: 5})

	iocs := a.Lookup("1.2.3.4")
	if len(iocs) != 2 {
		t.Errorf("attendu 2 IOC pour 1.2.3.4, got %d", len(iocs))
	}
}

// TestSources compte le nombre de sources distinctes.
func TestSources(t *testing.T) {
	a := New()
	a.Add(IOC{Value: "evil.com", Source: "abuse.ch"})
	a.Add(IOC{Value: "evil.com", Source: "otx"})
	a.Add(IOC{Value: "evil.com", Source: "otx"}) // doublon même source

	if got := a.Sources("evil.com"); got != 2 {
		t.Errorf("sources distinctes attendues 2, got %d", got)
	}
}

// TestNorm vérifie la normalisation (minuscules, trim).
func TestNorm(t *testing.T) {
	a := New()
	a.Add(IOC{Value: "  ABC.COM  "})
	if len(a.Lookup("abc.com")) != 1 {
		t.Error("la normalisation devrait permettre le lookup insensible à la casse")
	}
}

// TestStats vérifie le résumé.
func TestStats(t *testing.T) {
	a := New()
	a.Add(IOC{Value: "x", Source: "s1"})
	a.Add(IOC{Value: "y", Source: "s1"})
	stats := a.Stats()
	if stats["total"] != 2 {
		t.Errorf("total attendu 2, got %d", stats["total"])
	}
	if stats["by_source:s1"] != 2 {
		t.Errorf("by_source:s1 attendu 2, got %d", stats["by_source:s1"])
	}
}

// TestParseThreatFoxType vérifie le mapping des types ThreatFox.
func TestParseThreatFoxType(t *testing.T) {
	tests := []struct {
		tfType, value, wantVal string
		wantType               IOCType
	}{
		{"ip:port", "1.2.3.4:80", "1.2.3.4", IOCIP},
		{"domain", "evil.com", "evil.com", IOCDomain},
		{"sha256_hash", "ABCDEF123456", "abcdef123456", IOCHash},
		{"unknown", "x", "", ""},
	}
	for _, tc := range tests {
		typ, val := parseThreatFoxType(tc.tfType, tc.value)
		if typ != tc.wantType || val != tc.wantVal {
			t.Errorf("parseThreatFoxType(%q,%q) = (%q,%q), want (%q,%q)",
				tc.tfType, tc.value, typ, val, tc.wantType, tc.wantVal)
		}
	}
}
