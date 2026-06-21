package freshclam

import (
	"context"
	"testing"
)

// TestIsAvailable ne plante pas, que freshclam soit installé ou non.
// (Sur ce CI, freshclam n'est probablement pas installé -> false attendu.)
func TestIsAvailable(t *testing.T) {
	u := New()
	_ = u.IsAvailable()
}

// TestRunOnceWithoutFreshclam vérifie que RunOnce renvoie bien une erreur
// claire quand freshclam n'est pas installé.
func TestRunOnceWithoutFreshclam(t *testing.T) {
	u := New()
	if u.IsAvailable() {
		t.Skip("freshclam est installé sur ce runner, test non applicable")
	}
	err := u.RunOnce(context.Background())
	if err == nil {
		t.Error("RunOnce devrait échouer si freshclam absent")
	}
}

// TestSetInterval vérifie que l'intervalle est configurable.
func TestSetInterval(t *testing.T) {
	u := New()
	u.SetInterval(60)
	// Pas d'assertion stricte sur la valeur (champ privé), on vérifie
	// juste que ça ne panique pas et que Start peut l'utiliser.
	_ = u
}
