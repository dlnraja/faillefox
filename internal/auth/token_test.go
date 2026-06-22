package auth

import (
	"path/filepath"
	"testing"
)

// TestLoadOrCreateGenerates vérifie qu'un nouveau token est généré si absent.
func TestLoadOrCreateGenerates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	tok, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tok.Value()) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("token longueur attendue 64, got %d", len(tok.Value()))
	}
}

// TestLoadOrCreateReuse vérifie qu'un token existant est réutilisé.
func TestLoadOrCreateReuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	tok1, _ := LoadOrCreate(path)
	tok2, _ := LoadOrCreate(path)
	if tok1.Value() != tok2.Value() {
		t.Error("le token devrait être réutilisé entre 2 appels")
	}
}

// TestValidateCorrect vérifie qu'un token correct est validé.
func TestValidateCorrect(t *testing.T) {
	tok := &Token{value: "abc123"}
	if !tok.Validate("abc123") {
		t.Error("le token correct devrait être validé")
	}
}

// TestValidateIncorrect vérifie qu'un faux token est rejeté.
func TestValidateIncorrect(t *testing.T) {
	tok := &Token{value: "abc123"}
	if tok.Validate("wrong") {
		t.Error("un faux token ne devrait pas être validé")
	}
}

// TestValidateEmpty vérifie que les tokens vides sont rejetés.
func TestValidateEmpty(t *testing.T) {
	tok := &Token{value: "abc123"}
	if tok.Validate("") {
		t.Error("un token vide ne devrait pas être validé")
	}
	if (&Token{}).Validate("abc") {
		t.Error("un token nil ne devrait rien valider")
	}
}

// TestValidateTimingSafe vérifie qu'un token de longueur différente est rejeté
// sans panic (subtle.ConstantTimeCompare exige même longueur).
func TestValidateTimingSafe(t *testing.T) {
	tok := &Token{value: "abc123"}
	if tok.Validate("abc1234") { // longueur différente
		t.Error("un token de longueur différente ne devrait pas être validé")
	}
}
