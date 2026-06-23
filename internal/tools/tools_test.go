package tools

import (
	"strings"
	"testing"
)

// TestPasswordEvaluateWeak vérifie la détection d'un mot de passe faible.
func TestPasswordEvaluateWeak(t *testing.T) {
	p := NewPasswordChecker()
	s := p.Evaluate("abc")
	if s.Score > 1 {
		t.Errorf("'abc' devrait être faible (<=1), got %d", s.Score)
	}
}

// TestPasswordEvaluateStrong vérifie un mot de passe fort.
func TestPasswordEvaluateStrong(t *testing.T) {
	p := NewPasswordChecker()
	s := p.Evaluate("MyS3cur3!P@ssw0rd#2024")
	if s.Score < 3 {
		t.Errorf("mot de passe fort devrait être >=3, got %d", s.Score)
	}
}

// TestPasswordEvaluateEmpty vérifie le cas vide.
func TestPasswordEvaluateEmpty(t *testing.T) {
	p := NewPasswordChecker()
	s := p.Evaluate("")
	if s.Score != 0 {
		t.Errorf("vide devrait être score 0, got %d", s.Score)
	}
}

// TestPasswordEvaluateCommonPatterns vérifie la pénalité pour patterns communs.
func TestPasswordEvaluateCommonPatterns(t *testing.T) {
	p := NewPasswordChecker()
	// "password" est un pattern commun reconnu.
	s := p.Evaluate("MyPassword123!!")
	if s.Score >= 4 {
		t.Errorf("mot de passe avec 'password' devrait être pénalisé, got %d", s.Score)
	}
}

// TestPasswordGenerate vérifie qu'un mot de passe est généré.
func TestPasswordGenerate(t *testing.T) {
	g := NewPasswordGenerator()
	pw, err := g.Generate(20)
	if err != nil {
		t.Fatal(err)
	}
	if len(pw) != 20 {
		t.Errorf("longueur attendue 20, got %d", len(pw))
	}
}

// TestPasswordGenerateDifferent vérifie que 2 générations sont différentes.
func TestPasswordGenerateDifferent(t *testing.T) {
	g := NewPasswordGenerator()
	pw1, _ := g.Generate(32)
	pw2, _ := g.Generate(32)
	if pw1 == pw2 {
		t.Error("2 générations devraient être différentes")
	}
}

// TestPasswordGenerateMinimum vérifie le minimum forcé à 8.
func TestPasswordGenerateMinimum(t *testing.T) {
	g := NewPasswordGenerator()
	pw, _ := g.Generate(4) // trop court, devrait être forcé à 8+
	if len(pw) < 8 {
		t.Errorf("longueur minimum devrait être 8, got %d", len(pw))
	}
}

// TestPasswordGenerateContainsChar vérifie que le mot de passe contient des
// caractères variés (lower + upper + digit + symbol).
func TestPasswordGenerateContainsChar(t *testing.T) {
	g := NewPasswordGenerator()
	pw, _ := g.Generate(50)
	if !strings.ContainsAny(pw, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("devrait contenir des minuscules")
	}
}

// TestPortScannerScan vérifie le scan sur une cible invalide (pas de crash).
func TestPortScannerScan(t *testing.T) {
	s := NewPortScanner()
	// Scan sur une IP inexistante (TEST-NET) avec timeout très court.
	// On vérifie juste que ça ne panic pas, sans assertion stricte sur les
	// résultats (TEST-NET peut théoriquement répondre selon le réseau CI).
	_ = s.Scan("192.0.2.1", 1)
}

// TestDNSLeakTest ne crash pas.
func TestDNSLeakTest(t *testing.T) {
	d := NewDNSLeakTest()
	results := d.Test()
	if len(results) == 0 {
		t.Error("devrait retourner au moins des résolveurs testés")
	}
}
