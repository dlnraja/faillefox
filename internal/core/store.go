package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Store gère la persistance des règles et de la politique par défaut.
// En v1 il s'agit d'un simple fichier JSON lisible/éditable à la main.
type Store interface {
	Load() (rules []Rule, defaults Decision, err error)
	Save(rules []Rule, defaults Decision) error
}

// jsonStore est une implémentation fichier de Store.
type jsonStore struct {
	path string
}

type storePayload struct {
	Rules    []Rule    `json:"rules"`
	Defaults Decision  `json:"default"`
}

// NewFileStore crée un store basé sur un fichier JSON au chemin donné.
func NewFileStore(path string) Store {
	return &jsonStore{path: path}
}

func (s *jsonStore) Load() ([]Rule, Decision, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Première exécution : rien à charger, defaults = "ask".
			return nil, DecisionAsk, nil
		}
		return nil, "", err
	}
	var p storePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, "", err
	}
	if p.Defaults == "" {
		p.Defaults = DecisionAsk
	}
	if p.Rules == nil {
		p.Rules = []Rule{}
	}
	return p.Rules, p.Defaults, nil
}

func (s *jsonStore) Save(rules []Rule, defaults Decision) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	p := storePayload{Rules: rules, Defaults: defaults}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// Écriture atomique : fichier temporaire puis renommage.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
