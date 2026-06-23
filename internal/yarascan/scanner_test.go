package yarascan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadRules vérifie le chargement de règles YARA simplifiées.
func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yar")
	// Règle YARA simplifiée valide.
	content := `rule test_malware {
		strings:
			$s1 = "malware_signature"
			$s2 = "evil_payload"
		condition:
			any of them
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New()
	n, err := s.LoadRules(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("1 règle attendue, got %d", n)
	}
	if s.RuleCount() != 1 {
		t.Errorf("RuleCount attendu 1, got %d", s.RuleCount())
	}
}

// TestScanFileMatch vérifie qu'un fichier contenant une signature est détecté.
func TestScanFileMatch(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.yar")
	if err := os.WriteFile(rulePath, []byte(`rule detect_test {
		strings:
			$s1 = "EVIL_MARKER_12345"
		condition:
			$s1
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "malware.bin")
	if err := os.WriteFile(target, []byte("This file contains EVIL_MARKER_12345 payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := New()
	if _, err := s.LoadRules(rulePath); err != nil {
		t.Fatal(err)
	}
	matches, err := s.ScanFile(context.TODO(), target)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Error("devrait détecter le match")
	}
}

// TestScanFileNoMatch vérifie qu'un fichier sain ne matche pas.
func TestScanFileNoMatch(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.yar")
	if err := os.WriteFile(rulePath, []byte(`rule detect_test {
		strings:
			$s1 = "MALWARE_SIGNATURE_HERE"
		condition:
			$s1
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "clean.txt")
	if err := os.WriteFile(target, []byte("This is a perfectly clean file"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := New()
	if _, err := s.LoadRules(rulePath); err != nil {
		t.Fatal(err)
	}
	matches, err := s.ScanFile(context.TODO(), target)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("fichier sain ne devrait pas matcher, got %d matches", len(matches))
	}
}

// TestScanNoRules vérifie l'erreur quand aucune règle n'est chargée.
func TestScanNoRules(t *testing.T) {
	s := New()
	target := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(target, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := s.ScanFile(context.TODO(), target)
	if err == nil {
		t.Error("devrait échouer sans règles chargées")
	}
}

// TestIsAvailable vérifie la disponibilité.
func TestIsAvailable(t *testing.T) {
	s := New()
	if s.IsAvailable() {
		t.Error("nouveau scanner sans règles ne devrait pas être disponible")
	}
}

// TestLoadRulesMissingFile vérifie l'erreur sur fichier absent.
func TestLoadRulesMissingFile(t *testing.T) {
	s := New()
	_, err := s.LoadRules("/nonexistent/path/rules.yar")
	if err == nil {
		t.Error("fichier absent devrait être une erreur")
	}
}
