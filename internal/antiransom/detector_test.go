package antiransom

import (
	"path/filepath"
	"testing"
	"time"
)

// TestRansomNoteDetection vérifie la détection d'un fichier de rançon.
func TestRansomNoteDetection(t *testing.T) {
	var got *Alert
	d := New(func(a Alert) { got = &a })
	d.OnFileEvent("/tmp/Documents/README_DECRYPT.txt")
	if got == nil {
		t.Fatal("une ransom note devrait déclencher une alerte")
	}
	if got.Type != SeverityCritical {
		t.Errorf("sévérité attendue critical, got %s", got.Type)
	}
}

// TestRansomNoteVariants vérifie plusieurs patterns connus.
func TestRansomNoteVariants(t *testing.T) {
	cases := []string{
		"how_to_decrypt.html",
		"!RESTORE_FILES.txt",
		"lockbit_readme.txt",
	}
	for _, name := range cases {
		var got *Alert
		d := New(func(a Alert) { got = &a })
		d.OnFileEvent(filepath.Join(t.TempDir(), name))
		if got == nil {
			t.Errorf("%s aurait dû déclencher une alerte", name)
		}
	}
}

// TestEncryptedExtensionDetection vérifie la détection d'extensions chiffrées.
func TestEncryptedExtensionDetection(t *testing.T) {
	var got *Alert
	d := New(func(a Alert) { got = &a })
	d.OnFileEvent("/tmp/report.lockbit")
	if got == nil {
		t.Fatal("extension .lockbit aurait dû déclencher une alerte")
	}
	if got.Type != SeverityWarning {
		t.Errorf("sévérité attendue warning, got %s", got.Type)
	}
}

// TestNormalFileNoAlert vérifie qu'un fichier normal ne déclenche rien.
func TestNormalFileNoAlert(t *testing.T) {
	var got *Alert
	d := New(func(a Alert) { got = &a })
	d.OnFileEvent("/tmp/report.pdf")
	if got != nil {
		t.Errorf("un fichier normal ne devrait pas déclencher d'alerte: %+v", got)
	}
}

// TestRateLimiting vérifie le déclenchement au seuil d'écriture massive.
func TestRateLimiting(t *testing.T) {
	dir := t.TempDir()
	var alerts []Alert
	d := New(func(a Alert) { alerts = append(alerts, a) })
	d.SetProtectedDirs([]string{dir})
	d.SetThreshold(5, time.Minute) // seuil bas pour le test

	// Simule 6 écritures dans le dossier sensible (dépasse le seuil de 5).
	for i := 0; i < 6; i++ {
		d.OnFileEvent(filepath.Join(dir, "file.txt"))
	}
	if len(alerts) == 0 {
		t.Fatal("une alerte de rate anormal aurait dû être déclenchée")
	}
	// La dernière alerte doit être CRITICAL (chiffrement probable).
	last := alerts[len(alerts)-1]
	if last.Type != SeverityCritical {
		t.Errorf("sévérité attendue critical, got %s", last.Type)
	}
}

// TestRateLimitingReset vérifie que le compteur se reset après une alerte
// (pas de rafale d'alertes).
func TestRateLimitingReset(t *testing.T) {
	dir := t.TempDir()
	var count int
	d := New(func(a Alert) {
		if a.Type == SeverityCritical {
			count++
		}
	})
	d.SetProtectedDirs([]string{dir})
	d.SetThreshold(3, time.Minute)

	// 6 écritures > seuil 3 -> 1 seule alerte CRITICAL (pas 3).
	for i := 0; i < 6; i++ {
		d.OnFileEvent(filepath.Join(dir, "f.txt"))
	}
	if count != 1 {
		t.Errorf("1 alerte CRITICAL attendue (reset anti-rafale), got %d", count)
	}
}

// TestProtectedDirs vérifie la gestion des dossiers sensibles.
func TestProtectedDirs(t *testing.T) {
	d := New(func(a Alert) {})
	custom := []string{"/home/me/important"}
	d.SetProtectedDirs(custom)
	got := d.ProtectedDirs()
	if len(got) != 1 || got[0] != "/home/me/important" {
		t.Errorf("ProtectedDirs: %v, want %v", got, custom)
	}
}

// TestIsProtected vérifie la détection d'appartenance à un dossier protégé.
func TestIsProtected(t *testing.T) {
	dir := t.TempDir()
	d := New(func(a Alert) {})
	d.SetProtectedDirs([]string{dir})
	if !d.isProtected(filepath.Join(dir, "subdir")) {
		t.Error("un sous-dossier d'un dossier protégé devrait être protégé")
	}
	if d.isProtected("/tmp/elsewhere") {
		t.Error("un dossier hors protection ne devrait pas être protégé")
	}
}
