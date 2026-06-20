package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileStoreRoundTrip vérifie qu'une sauvegarde puis un rechargement
// restitue fidèlement règles et politique par défaut.
func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "policies.json"))

	in := []Rule{
		{ID: "r1", AppID: "a.exe", Action: DecisionDeny, Port: 443},
		{ID: "r2", AppID: "b.exe", Action: DecisionAllow},
	}
	if err := store.Save(in, DecisionAsk); err != nil {
		t.Fatal(err)
	}

	out, defaults, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if defaults != DecisionAsk {
		t.Errorf("default = %v, want ask", defaults)
	}
	if len(out) != 2 {
		t.Fatalf("nb règles = %d, want 2", len(out))
	}
	if out[0].ID != "r1" || out[0].Action != DecisionDeny || out[0].Port != 443 {
		t.Errorf("règle r1 mal restaurée: %+v", out[0])
	}
}

// TestFileStoreMissingFile vérifie le comportement à la première exécution
// (fichier absent) : pas d'erreur, defaults = "ask", règles vides.
func TestFileStoreMissingFile(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "absent.json"))

	rules, defaults, err := store.Load()
	if err != nil {
		t.Fatalf("un fichier absent ne doit pas être une erreur: %v", err)
	}
	if defaults != DecisionAsk {
		t.Errorf("defaults = %v, want ask", defaults)
	}
	if len(rules) != 0 {
		t.Errorf("règles attendues vides, got %d", len(rules))
	}
}

// TestFileStoreCreatesParentDir vérifie que le store crée les répertoires
// parents s'ils n'existent pas (cas ~/.faillefox/ à la première exécution).
func TestFileStoreCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sous", "dossier", "policies.json")
	store := NewFileStore(nested)

	if err := store.Save([]Rule{{ID: "x", Action: DecisionDeny}}, DecisionAllow); err != nil {
		t.Fatalf("la création du répertoire parent a échoué: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("le fichier n'a pas été créé: %v", err)
	}
}
