package settings

import (
	"path/filepath"
	"testing"
)

// TestDefault vérifie que les défauts sont sûrs.
func TestDefault(t *testing.T) {
	s := Default()
	if !s.Firewall {
		t.Error("le pare-feu devrait être activé par défaut")
	}
	if !s.DNSSinkhole {
		t.Error("le DNS sinkhole devrait être activé par défaut")
	}
	if !s.LoopbackOnly {
		t.Error("loopback only devrait TOUJOURS être true (non négociable)")
	}
	if s.UIMode != UIModeSimple {
		t.Error("le mode par défaut devrait être simple")
	}
}

// TestLoadAbsent vérifie le comportement sur fichier absent (1ʳᵉ exécution).
func TestLoadAbsent(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("fichier absent ne devrait pas être une erreur: %v", err)
	}
	if !s.Firewall {
		t.Error("fichier absent -> défauts, firewall devrait être true")
	}
}

// TestLoadCorrupt vérifie le repli sur défauts si JSON corrompu.
func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	// Écrit un JSON invalide.
	_ = writeFile(path, "{not valid json")
	s, err := Load(path)
	if err != nil {
		t.Errorf("fichier corrompu ne devrait pas propager l'erreur: %v", err)
	}
	// Doit retomber sur les défauts.
	if !s.Firewall {
		t.Error("fichier corrompu -> défauts attendus")
	}
}

// TestSaveLoadRoundTrip vérifie la persistance.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Default()
	s.path = path
	s.AntiRansomware = true
	s.Theme = "light"
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.AntiRansomware {
		t.Error("anti_ransomware non restauré")
	}
	if loaded.Theme != "light" {
		t.Errorf("thème attendu light, got %s", loaded.Theme)
	}
}

// TestUpdatePatch vérifie la mise à jour partielle.
func TestUpdatePatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Default()
	s.path = path
	// Patch : désactive gamification, change le thème.
	if err := s.Update(map[string]any{
		"gamification": false,
		"theme":        "auto",
	}); err != nil {
		t.Fatal(err)
	}
	if s.Gamification {
		t.Error("gamification devrait être désactivée après patch")
	}
	if s.Theme != "auto" {
		t.Errorf("thème attendu auto, got %s", s.Theme)
	}
	// Les autres champs doivent être conservés.
	if !s.Firewall {
		t.Error("firewall non patché devrait rester true")
	}
}

// TestUpdateLoopbackUnnegotiable vérifie qu'un client ne peut PAS désactiver
// loopback_only via un patch.
func TestUpdateLoopbackUnnegotiable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Default()
	s.path = path
	// Un client malveillant tente de désactiver loopback.
	_ = s.Update(map[string]any{"loopback_only": false})
	if !s.LoopbackOnly {
		t.Error("loopback_only doit rester true même si le patch tente false (non négociable)")
	}
}

// writeFile helper (évite d'importer os juste pour ça dans le test).
func writeFile(path, content string) error {
	return writeFileImpl(path, content)
}
