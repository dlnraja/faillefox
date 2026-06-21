// Package logging fournit un journal persistant avec rotation, pour garder
// une trace des décisions du pare-feu même après redémarrage.
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dlnraja/faillefox/internal/core"
)

// RotatingLogger écrit les événements du pare-feu dans un fichier JSONL
// (une ligne JSON par événement). Le fichier est automatiquement roté quand
// il dépasse MaxBytes ; on conserve MaxFiles anciens fichiers.
type RotatingLogger struct {
	mu       sync.Mutex
	dir      string
	prefix   string // ex "events"
	MaxBytes int64
	MaxFiles int

	f    *os.File
	size int64
}

// NewRotatingLogger crée un logger dans dir, préfixe "events".
func NewRotatingLogger(dir, prefix string) (*RotatingLogger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	l := &RotatingLogger{
		dir:      dir,
		prefix:   prefix,
		MaxBytes: 5 * 1024 * 1024, // 5 Mo par fichier
		MaxFiles: 3,               // 3 archives : ~15 Mo max
	}
	if err := l.openCurrent(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *RotatingLogger) openCurrent() error {
	path := filepath.Join(l.dir, l.prefix+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, _ := f.Stat()
	l.f = f
	if info != nil {
		l.size = info.Size()
	}
	return nil
}

// Write écrit un événement + rotation si nécessaire. Implémente le contrat
// attendu par Engine via une fonction de callback.
func (l *RotatingLogger) Write(ev core.Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	line := append(data, '\n')
	n, err := l.f.Write(line)
	if err != nil {
		return err
	}
	l.size += int64(n)

	if l.size >= l.MaxBytes {
		l.rotate()
	}
	return nil
}

// rotate ferme le fichier courant, le renomme avec un timestamp, supprime
// les archives trop anciennes, et ouvre un nouveau fichier.
func (l *RotatingLogger) rotate() {
	if l.f != nil {
		_ = l.f.Close()
	}
	stamp := time.Now().Format("20060102-150405")
	old := filepath.Join(l.dir, l.prefix+".jsonl")
	archived := filepath.Join(l.dir, fmt.Sprintf("%s-%s.jsonl", l.prefix, stamp))
	_ = os.Rename(old, archived)

	// Nettoyage des archives trop anciennes (on garde MaxFiles).
	l.pruneArchives()
	_ = l.openCurrent()
}

func (l *RotatingLogger) pruneArchives() {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}
	var archs []string
	for _, e := range entries {
		name := e.Name()
		if name != l.prefix+".jsonl" && len(name) > len(l.prefix) && name[:len(l.prefix)] == l.prefix {
			archs = append(archs, filepath.Join(l.dir, name))
		}
	}
	// On supprime les plus anciens au-delà de MaxFiles.
	for len(archs) > l.MaxFiles {
		_ = os.Remove(archs[0])
		archs = archs[1:]
	}
}

// Close ferme proprement le logger.
func (l *RotatingLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		return l.f.Close()
	}
	return nil
}

// Compile-time check : RotatingLogger exposes Write(io.Writer-like usage).
var _ io.Closer = (*RotatingLogger)(nil)
