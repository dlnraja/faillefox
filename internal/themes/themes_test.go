package themes

import "testing"

// TestList vérifie que les 3 thèmes sont proposés.
func TestList(t *testing.T) {
	list := List()
	if len(list) != 3 {
		t.Errorf("3 thèmes attendus, got %d", len(list))
	}
	ids := map[Theme]bool{}
	for _, th := range list {
		ids[th.ID] = true
	}
	for _, want := range []Theme{ThemeDark, ThemeLight, ThemeAuto} {
		if !ids[want] {
			t.Errorf("thème %s manquant dans List()", want)
		}
	}
}

// TestIsValid vérifie la reconnaissance des thèmes.
func TestIsValid(t *testing.T) {
	valid := []Theme{ThemeDark, ThemeLight, ThemeAuto}
	for _, th := range valid {
		if !IsValid(th) {
			t.Errorf("%s devrait être valide", th)
		}
	}
	if IsValid("invalid") {
		t.Error("'invalid' ne devrait pas être valide")
	}
}

// TestNormalize vérifie la normalisation vers le défaut.
func TestNormalize(t *testing.T) {
	if Normalize("invalid") != Default {
		t.Error("thème invalide devrait être normalisé en Default")
	}
	if Normalize(ThemeLight) != ThemeLight {
		t.Error("thème valide devrait être renvoyé tel quel")
	}
}

// TestDefault vérifie que le défaut est sombre.
func TestDefault(t *testing.T) {
	if Default != ThemeDark {
		t.Errorf("défaut attendu sombre, got %s", Default)
	}
}
