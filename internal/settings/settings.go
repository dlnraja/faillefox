// Package settings gère la configuration utilisateur persistée de Faillefox.
//
// Toutes les options sont centralisées ici, avec deux niveaux d'interface :
//   - Mode SIMPLE : quelques interrupteurs globaux (protection auto, thème,
//     niveau de notification).
//   - Mode AVANCÉ : accès fin à chaque module (DNS, CVE, threat intel,
//     ClamAV, YARA, anti-ransomware, gamification, scheduler, profils).
//
// Persistance : fichier JSON (~/.faillefox/settings.json), écriture atomique.
// En cas de fichier absent/corrompu, on retombe sur les valeurs par défaut.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UIMode est le niveau de détail de l'interface.
type UIMode string

const (
	UIModeSimple  UIMode = "simple"  // vue épurée, interrupteurs globaux
	UIModeAdvanced UIMode = "advanced" // accès fin à chaque module
)

// NotificationLevel contrôle la verbosité des alertes.
type NotificationLevel string

const (
	NotifyMinimal NotificationLevel = "minimal" // alertes critiques seulement
	NotifyNormal  NotificationLevel = "normal"  // défaut
	NotifyVerbose NotificationLevel = "verbose" // tout
)

// Settings est la configuration complète persistée.
type Settings struct {
	mu sync.Mutex `json:"-"`

	// --- Mode simple (globaux) ---
	UIMode           UIMode             `json:"ui_mode"`            // simple | advanced
	Theme            string             `json:"theme"`              // dark | light | auto
	Notifications    NotificationLevel  `json:"notifications"`      // minimal|normal|verbose
	AutoProtect      bool               `json:"auto_protect"`       // active toutes les protections recommandées

	// --- Modules (mode avancé) ---
	Firewall         bool               `json:"firewall"`
	DNSSinkhole      bool               `json:"dns_sinkhole"`
	AntiAds          bool               `json:"anti_ads"`
	AntiTrackers     bool               `json:"anti_trackers"`
	AntiMalware      bool               `json:"anti_malware"`
	AntiPhishing     bool               `json:"anti_phishing"`
	AntiRansomware   bool               `json:"anti_ransomware"`
	AVScanner        bool               `json:"av_scanner"`         // ClamAV
	YARAScanner      bool               `json:"yara_scanner"`
	CVEFeed          bool               `json:"cve_feed"`
	ThreatIntel      bool               `json:"threat_intel"`
	Gamification     bool               `json:"gamification"`

	// --- Réseau ---
	Profile          string             `json:"profile"`            // home | office | public
	LoopbackOnly     bool               `json:"loopback_only"`      // toujours true (sécurité)

	// --- Automatisation ---
	AutoUpdate       bool               `json:"auto_update"`
	UpdateInterval   string             `json:"update_interval"`    // durée, ex "6h"
	Freshclam        bool               `json:"freshclam"`

	// --- Méta ---
	Version          string             `json:"settings_version"`
	UpdatedAt        time.Time          `json:"updated_at"`

	path string `json:"-"`
}

// Default renvoie les paramètres par défaut (recommandations sûres).
func Default() *Settings {
	return &Settings{
		UIMode:         UIModeSimple,
		Theme:          "dark",
		Notifications:  NotifyNormal,
		AutoProtect:    true,
		Firewall:       true,
		DNSSinkhole:    true,
		AntiAds:        true,
		AntiTrackers:   true,
		AntiMalware:    true,
		AntiPhishing:   true,
		AntiRansomware: false, // opt-in (fsnotify peut être bruyant)
		AVScanner:      false, // opt-in (nécessite ClamAV installé)
		YARAScanner:    false,
		CVEFeed:        true,
		ThreatIntel:    true,
		Gamification:   true,
		Profile:        "home",
		LoopbackOnly:   true, // non négociable
		AutoUpdate:     true,
		UpdateInterval: "6h",
		Freshclam:      false,
		Version:        "1",
	}
}

// Load charge les settings depuis path, ou Default() si absent/corrompu.
func Load(path string) (*Settings, error) {
	s := Default()
	s.path = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // première exécution : défauts
		}
		return s, err
	}
	// On décode PAR-DESSUS les défauts : les champs absents du fichier
	// conservent leur valeur par défaut (forward-compatibilité).
	loaded := Default()
	loaded.path = path
	if err := json.Unmarshal(data, loaded); err != nil {
		return s, nil // fichier corrompu -> défauts, pas de crash
	}
	// LoopbackOnly est non négociable : on le force à true.
	loaded.LoopbackOnly = true
	return loaded, nil
}

// Save persiste les settings (écriture atomique).
func (s *Settings) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Update applique des changements partiels (depuis l'UI) et persiste.
// Les champs fournis dans patch écrasent ceux de s ; les autres sont
// conservés. LoopbackOnly reste toujours true (non négociable).
func (s *Settings) Update(patch map[string]any) error {
	s.mu.Lock()
	// Marshalling du patch en JSON, puis unmarshal par-dessus s.
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if err := json.Unmarshal(patchJSON, s); err != nil {
		s.mu.Unlock()
		return err
	}
	s.LoopbackOnly = true // force, non négociable
	s.mu.Unlock()
	return s.Save()
}
