package antiransom

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// Watcher surveille activement les dossiers sensibles et transmet chaque
// événement fichier au Detector. C'est le pont entre fsnotify et la logique
// de détection comportementale.
//
// Attention performances : fsnotify peut générer beaucoup d'événements sur
// un dossier très actif (compilations, sync cloud). Le Detector fait le
// filtrage (rate-limiting), pas le Watcher — ce dernier ne fait que relayer.
type Watcher struct {
	detector *Detector
	fsn      *fsnotify.Watcher
}

// NewWatcher crée un watcher lié à un détecteur existant.
// Les dossiers surveillés sont ceux déclarés dans detector.ProtectedDirs().
func NewWatcher(detector *Detector) (*Watcher, error) {
	fsn, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}
	w := &Watcher{detector: detector, fsn: fsn}

	// Enregistre chaque dossier sensible (récursivement, 1 niveau).
	for _, dir := range detector.ProtectedDirs() {
		if err := w.addWatch(dir); err != nil {
			// Un dossier inexistant n'est pas fatal (ex: OneDrive absent).
			log.Printf("[antiransom] watch %s: %v (ignoré)", dir, err)
		}
	}
	return w, nil
}

// addWatch ajoute un dossier + ses sous-dossiers directs à fsnotify.
func (w *Watcher) addWatch(dir string) error {
	if err := w.fsn.Add(dir); err != nil {
		return err
	}
	// On descend d'un niveau pour couvrir les sous-dossiers courants
	// (/Documents/Travail, /Pictures/Vacances...). Pas de récursion
	// complète : ça grimpe vite en nombre de watches sur un gros disque.
	entries, err := readDirEntries(dir)
	if err != nil {
		return nil // le watch du parent suffit
	}
	for _, e := range entries {
		if e.IsDir() {
			_ = w.fsn.Add(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}

// Start lance la boucle d'écoute. Bloquant ; à lancer dans une goroutine.
// S'arrête proprement quand le contexte est annulé.
func (w *Watcher) Start(ctx context.Context) error {
	defer func() { _ = w.fsn.Close() }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-w.fsn.Events:
			if !ok {
				return nil
			}
			// On ne traite que les écritures/créations/renommages (pas les chmod).
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
				w.detector.OnFileEvent(event.Name)
			}
		case err, ok := <-w.fsn.Errors:
			if !ok {
				return nil
			}
			log.Printf("[antiransom] erreur watcher: %v", err)
		}
	}
}

// readDirEntries liste les entrées d'un dossier (helper, isolé pour les tests).
func readDirEntries(dir string) ([]dirEntry, error) {
	return readDirEntriesImpl(dir)
}

// dirEntry est une abstraction minimale d'une entrée de dossier.
type dirEntry interface {
	Name() string
	IsDir() bool
}
