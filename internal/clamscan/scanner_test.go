package clamscan

import "testing"

// TestParseClamdLineOK vérifie le parsing d'une réponse clamd "OK".
func TestParseClamdLineOK(t *testing.T) {
	r := parseClamdLine("/tmp/file.txt", "/tmp/file.txt: OK")
	if r.Infected {
		t.Error("fichier sain ne doit pas être marqué infecté")
	}
	if r.Path != "/tmp/file.txt" {
		t.Errorf("path: %s", r.Path)
	}
}

// TestParseClamdLineInfected vérifie le parsing d'une détection clamd.
func TestParseClamdLineInfected(t *testing.T) {
	r := parseClamdLine("/tmp/eicar.com", "/tmp/eicar.com: Eicar-Test-Signature FOUND")
	if !r.Infected {
		t.Error("fichier infecté devrait être détecté")
	}
	if r.Signature != "Eicar-Test-Signature" {
		t.Errorf("signature: %q, want Eicar-Test-Signature", r.Signature)
	}
}

// TestParseClamscanOutputInfected vérifie le parsing de clamscan CLI.
func TestParseClamscanOutputInfected(t *testing.T) {
	out := "/tmp/eicar.com: Eicar-Test-Signature FOUND\n\n----------- SCAN SUMMARY -----------\n"
	r := parseClamscanOutput("/tmp/eicar.com", out)
	if !r.Infected {
		t.Error("devrait être infecté")
	}
	if r.Signature != "Eicar-Test-Signature" {
		t.Errorf("signature: %q", r.Signature)
	}
}

// TestParseClamscanOutputClean vérifie le parsing d'un scan propre.
func TestParseClamscanOutputClean(t *testing.T) {
	out := "\n----------- SCAN SUMMARY -----------\nInfected files: 0\n"
	r := parseClamscanOutput("/tmp/file.txt", out)
	if r.Infected {
		t.Error("fichier sain ne doit pas être infecté")
	}
}

// TestScannerAvailability vérifie que IsAvailable ne panique pas.
// (Sur ce CI, ClamAV n'est probablement pas installé -> false attendu.)
func TestScannerAvailability(t *testing.T) {
	s := New()
	// Pas d'assertion stricte : on vérifie juste que ça ne plante pas.
	_ = s.IsAvailable()
}
