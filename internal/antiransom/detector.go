// Package antiransom détecte les comportements typiques d'un ransomware et
// alerte l'utilisateur. Ce n'est PAS un moteur heuristique ML (impossible
// seul) — c'est une détection comportementale par règles simples mais
// efficaces :
//
//   1. Surveillance des DOSSIERS SENSIBLES (Documents, Photos, Bureau...).
//      Si un processus inconnu y écrit massivement, alerte.
//   2. Détection des RANSOM NOTES (fichiers du type "README_DECRYPT.txt",
//      "HOW_TO_DECRYPT.html", "!RESTORE_FILES.txt"...).
//   3. Détection des EXTENSIONS CHIFFRÉES connues (.locked, .crypto,
//      .encrypted, .ryuk, .conty, .lockbit...).
//   4. RATE-LIMITING : seuil d'écriture anormale (ex: >200 fichiers modifiés
//      en 30s dans un dossier sensible = comportement de chiffrement).
//
// Limitations honnêtes :
//   - Ne peut PAS identifier un ransomware zero-day inconnu par sa signature.
//   - Ne restaure PAS les fichiers (nécessite backups/shadow copies).
//   - La surveillance par fsnotify peut manquer des événements si le volume
//     est très élevé (rationnel : on alerte dès le seuil, même sur un sous-ensemble).
//   - Ne remplace pas une vraie solution EDR commerciale.
package antiransom

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AlertSeverity est la sévérité d'une alerte anti-ransomware.
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"  // activité suspecte (à surveiller)
	SeverityCritical AlertSeverity = "critical" // comportement de chiffrement probable
)

// Alert signale une activité potentiellement liée à un ransomware.
type Alert struct {
	Type        AlertSeverity `json:"type"`
	Title       string        `json:"title"`
	Path        string        `json:"path"`
	Description string        `json:"description"`
	At          time.Time     `json:"at"`
}

// Detector surveille l'activité fichier et détecte les comportements suspects.
type Detector struct {
	mu sync.Mutex

	// Dossiers sensibles surveillés (Documents, Photos...).
	protectedDirs []string

	// Fenêtre glissante : nombre de fichiers modifiés récemment, par dossier.
	// Permet de détecter le rate anormal (chiffrement = milliers de fichiers/sec).
	recentWrites map[string][]time.Time // clé: dir -> timestamps

	// Seuil d'alerte : si >threshold fichiers modifiés dans window secondes
	// dans un dossier sensible, on déclenche une alerte CRITICAL.
	threshold int
	window    time.Duration

	// Callback appelé à chaque alerte (ex: journalisation, notification UI).
	onAlert func(Alert)
}

// New crée un détecteur avec les seuils par défaut.
//   - threshold = 200 fichiers modifiés
//   - window = 30 secondes
func New(onAlert func(Alert)) *Detector {
	return &Detector{
		protectedDirs: defaultProtectedDirs(),
		recentWrites:  make(map[string][]time.Time),
		threshold:     200,
		window:        30 * time.Second,
		onAlert:       onAlert,
	}
}

// SetProtectedDirs définit les dossiers sensibles surveillés.
func (d *Detector) SetProtectedDirs(dirs []string) {
	d.mu.Lock()
	d.protectedDirs = dirs
	d.mu.Unlock()
}

// ProtectedDirs renvoie la liste des dossiers protégés.
func (d *Detector) ProtectedDirs() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.protectedDirs))
	copy(out, d.protectedDirs)
	return out
}

// SetThreshold ajuste le seuil de détection (fichiers / fenêtre).
func (d *Detector) SetThreshold(threshold int, window time.Duration) {
	d.mu.Lock()
	d.threshold = threshold
	d.window = window
	d.mu.Unlock()
}

// OnFileEvent est appelé pour chaque modification de fichier observée
// (création, écriture, renommage). C'est l'entrée principale du détecteur.
// path est le chemin absolu du fichier concerné.
func (d *Detector) OnFileEvent(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Détection de ransom note (nom de fichier caractéristique).
	if alert := d.checkRansomNote(path); alert != nil {
		d.fire(*alert)
	}

	// 2. Détection d'extension chiffrée connue.
	if alert := d.checkEncryptedExt(path); alert != nil {
		d.fire(*alert)
	}

	// 3. Rate-limiting : compte les écritures récentes par dossier sensible.
	dir := filepath.Dir(path)
	if d.isProtected(dir) {
		d.recentWrites[dir] = append(d.recentWrites[dir], time.Now())
		d.pruneOld(dir)
		if len(d.recentWrites[dir]) > d.threshold {
			d.fire(Alert{
				Type:        SeverityCritical,
				Title:       "Activité de chiffrement massive détectée",
				Path:        dir,
				Description: "Trop de fichiers modifiés en peu de temps dans un dossier sensible — comportement typique d'un ransomware",
				At:          time.Now(),
			})
			// On reset le compteur pour éviter les alertes en rafale.
			d.recentWrites[dir] = nil
		}
	}
}

// checkRansomNote détecte les noms de fichiers laissés par les ransomwares.
func (d *Detector) checkRansomNote(path string) *Alert {
	name := strings.ToLower(filepath.Base(path))
	for _, pattern := range ransomNotePatterns {
		if strings.Contains(name, pattern) {
			return &Alert{
				Type:        SeverityCritical,
				Title:       "Ransom note détectée",
				Path:        path,
				Description: "Un fichier typique des rançongiciels (« " + filepath.Base(path) + " ») a été créé — possible infection",
				At:          time.Now(),
			}
		}
	}
	return nil
}

// checkEncryptedExt détecte les extensions de fichiers chiffrés connues.
func (d *Detector) checkEncryptedExt(path string) *Alert {
	ext := strings.ToLower(filepath.Ext(path))
	for _, known := range encryptedExtensions {
		if ext == known {
			return &Alert{
				Type:        SeverityWarning,
				Title:       "Extension chiffrée détectée",
				Path:        path,
				Description: "Le fichier porte l'extension « " + ext + " » associée à des ransomwares connus",
				At:          time.Now(),
			}
		}
	}
	return nil
}

// isProtected vérifie si un dossier est sous surveillance.
func (d *Detector) isProtected(dir string) bool {
	abs, _ := filepath.Abs(dir)
	for _, p := range d.protectedDirs {
		pAbs, _ := filepath.Abs(p)
		if strings.HasPrefix(abs, pAbs) {
			return true
		}
	}
	return false
}

// pruneOld retire les timestamps plus vieux que la fenêtre.
func (d *Detector) pruneOld(dir string) {
	cutoff := time.Now().Add(-d.window)
	recent := d.recentWrites[dir][:0]
	for _, t := range d.recentWrites[dir] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	d.recentWrites[dir] = recent
}

// fire déclenche une alerte (callback utilisateur).
func (d *Detector) fire(a Alert) {
	if d.onAlert != nil {
		d.onAlert(a)
	}
}

// ---- données connues ------------------------------------------------------

// ransomNotePatterns : noms (ou fragments) typiques des fichiers laissés
// par les ransomwares pour exiger la rançon. Liste non exhaustive, basée
// sur les familles les plus actives (LockBit, Conti, Ryuk, BlackCat...).
var ransomNotePatterns = []string{
	"readme_decrypt", "how_to_decrypt", "restore_files", "ransom",
	"!restore", "decrypt_instruction", "recovery", "_readme.",
	"info.txt", "help_decrypt", "your_files", "lockbit", "conti",
	"ryuk_readme", "blackcat", "wallet", "blackmail",
}

// encryptedExtensions : extensions associées au chiffrement par ransomware.
// Source : compilation des familles LockBit, Conti, BlackCat, Royal, Akira...
var encryptedExtensions = []string{
	".locked", ".crypto", ".encrypted", ".enc", ".crypted",
	".lockbit", ".conti", ".ryuk", ".blackcat", ".royal",
	".akira", ".stop", ".mzqw", ".snatch", ".phobos",
	".onion", ".wallet", ".crypt", ".cry", ".ryp",
	".gesd", ".koaw", ".mpaj", ".npre", ".kiiu",
}
