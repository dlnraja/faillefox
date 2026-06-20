package core

import "context"

// Driver est l'interface que chaque backend natif doit implémenter.
//
// Le cœur ne fait jamais d'appels système lui-même : il délègue au Driver
// sélectionné à l'initialisation. Un Driver simule (stub) le filtrage,
// un autre pilote WFP sur Windows, un autre VPNService sur Android, etc.
type Driver interface {
	// Name identifie le backend ("stub", "windows-wfp",
	// "android-vpn", "linux-nftables").
	Name() string

	// Start met en place le filtrage et commence à appeler engine.Intercept
	// pour chaque connexion sortante détectée. Bloque jusqu'à ce que le
	// contexte soit annulé ou que Stop soit appelé.
	Start(ctx context.Context, engine *Engine) error

	// ListApps renvoie la liste des applications connues du système qui
	// sont susceptibles d'émettre du trafic réseau. Utilisé par l'UI.
	ListApps() ([]App, error)

	// ApplyRules demande au backend de (re)installer les règles dans le
	// mécanisme natif (WFP, nftables...). Peut être un no-op pour les
	// backends qui interrogent le moteur en temps réel.
	ApplyRules(rules []Rule) error

	// Stop arrête le filtrage et libère les ressources natives.
	Stop() error
}

// App est une application connue du système, exposée à l'UI.
type App struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	IconData string `json:"icon_data,omitempty"` // base64 PNG, optionnel
}

// DriverFactory construit un Driver à partir d'une config. Enregistré dans
// registry.go au moment du main, selon la plateforme cible.
type DriverFactory func(cfg DriverConfig) (Driver, error)

// DriverConfig regroupe les options communes à tous les backends.
type DriverConfig struct {
	// Le pilote à utiliser ("stub", "windows-wfp", "android-vpn",
	// "linux-nftables"). Défaut: "stub".
	Driver string `json:"driver"`

	// PollInterval est la fréquence de simulation du backend stub.
	PollInterval string `json:"poll_interval,omitempty"`
}
