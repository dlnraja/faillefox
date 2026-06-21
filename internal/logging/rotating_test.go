package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dlnraja/faillefox/internal/core"
)

// TestRotatingLoggerWrite vérifie qu'un événement est écrit au format JSONL.
func TestRotatingLoggerWrite(t *testing.T) {
	dir := t.TempDir()
	l, err := NewRotatingLogger(dir, "events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	ev := core.Event{
		ID:         "abc",
		Decision:   core.DecisionDeny,
		Reason:     "rule:1",
		Connection: core.Connection{AppID: "evil.exe"},
	}
	if err := l.Write(ev); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"id":"abc"`) {
		t.Errorf("événement mal écrit: %s", data)
	}
	// Vérifie que c'est du JSON valide (une ligne).
	var got core.Event
	if err := json.Unmarshal(data[:len(data)-1], &got); err != nil {
		t.Errorf("sortie non JSON: %v", err)
	}
}

// TestRotatingLoggerRotation vérifie la rotation quand le fichier dépasse
// MaxBytes. On force un MaxBytes très petit pour déclencher la rotation.
func TestRotatingLoggerRotation(t *testing.T) {
	dir := t.TempDir()
	l, err := NewRotatingLogger(dir, "events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	l.MaxBytes = 200 // seuil très bas pour forcer la rotation

	for i := 0; i < 20; i++ {
		if err := l.Write(core.Event{ID: "x", Connection: core.Connection{AppID: "app"}}); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Il devrait y avoir le fichier courant + au moins 1 archive datée.
	nFiles := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			nFiles++
		}
	}
	if nFiles < 2 {
		t.Errorf("rotation attendue (>=2 fichiers), got %d", nFiles)
	}
}
