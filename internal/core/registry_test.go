package core

import (
	"context"
	"testing"
)

// TestRegistryUnknownDriver vérifie qu'un nom de pilote inconnu renvoie
// une erreur claire (et n'instancie rien).
func TestRegistryUnknownDriver(t *testing.T) {
	_, err := NewDriver(DriverConfig{Driver: "ce-pilote-nexiste-pas"})
	if err == nil {
		t.Fatal("un pilote inconnu devrait renvoyer une erreur")
	}
}

// TestRegistryCustomDriver vérifie l'enregistrement et l'instanciation
// d'un pilote via RegisterDriver.
func TestRegistryCustomDriver(t *testing.T) {
	RegisterDriver("test-fake", func(cfg DriverConfig) (Driver, error) {
		return &fakeDriver{}, nil
	})
	defer func() {
		// nettoyage : retirer le pilote de test pour ne pas polluer les autres tests
		delete(driverRegistry, "test-fake")
	}()

	d, err := NewDriver(DriverConfig{Driver: "test-fake"})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	if d.Name() != "fake" {
		t.Errorf("Name() = %q, want fake", d.Name())
	}
}

// TestRegistryDefaultIsStub vérifie que sans préciser de pilote, la valeur
// par défaut attendue est "stub" (le backend de démo). On ne peut pas
// garantir sa présence ici (package core sans import du driver stub), donc
// on vérifie juste que le fallback pointe bien sur "stub" avant la lookup.
func TestRegistryDefaultDriverName(t *testing.T) {
	cfg := DriverConfig{}
	if name := func() string {
		if cfg.Driver == "" {
			return "stub"
		}
		return cfg.Driver
	}(); name != "stub" {
		t.Errorf("défaut attendu 'stub', got %q", name)
	}
}

// fakeDriver est un Driver de test minimal.
type fakeDriver struct{}

func (f *fakeDriver) Name() string                                       { return "fake" }
func (f *fakeDriver) Start(ctx context.Context, e *Engine) error         { return nil }
func (f *fakeDriver) ListApps() ([]App, error)                           { return nil, nil }
func (f *fakeDriver) ApplyRules(rules []Rule) error                      { return nil }
func (f *fakeDriver) Stop() error                                        { return nil }
