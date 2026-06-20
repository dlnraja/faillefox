// Package stub est un backend de démonstration qui simule des connexions
// réseau sortantes à intervalle régulier. Il permet de tester TOUT le
// pipeline (moteur, journal, API, UI) sans droits administrateur et sans
// aucune interception réelle du trafic.
//
// En production, c'est le driver windows-wfp / android-vpn / linux-nftables
// qui prend le relais et fait de VRAIES interceptions.
package stub

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/dlnraja/faillefox/internal/core"
)

func init() {
	core.RegisterDriver("stub", New)
}

// Driver implémente core.Driver pour la simulation.
type Driver struct {
	interval time.Duration
	apps     []core.App
	cancel   context.CancelFunc
}

// New construit un driver stub.
func New(cfg core.DriverConfig) (core.Driver, error) {
	interval := 3 * time.Second
	if cfg.PollInterval != "" {
		if d, err := time.ParseDuration(cfg.PollInterval); err == nil {
			interval = d
		}
	}
	return &Driver{
		interval: interval,
		apps: []core.App{
			{ID: "C:\\Windows\\System32\\chrome.exe", Name: "Google Chrome", Path: "C:\\Windows\\System32\\chrome.exe"},
			{ID: "C:\\Windows\\System32\\svchost.exe", Name: "Service hôte (svchost)", Path: "C:\\Windows\\System32\\svchost.exe"},
			{ID: "/usr/bin/curl", Name: "curl", Path: "/usr/bin/curl"},
			{ID: "com.android.chrome", Name: "Chrome (Android)", Path: "/data/app/com.android.chrome"},
		},
	}, nil
}

// Name identifie le pilote.
func (d *Driver) Name() string { return "stub" }

// ListApps renvoie les applications simulées.
func (d *Driver) ListApps() ([]core.App, error) {
	out := make([]core.App, len(d.apps))
	copy(out, d.apps)
	return out, nil
}

// ApplyRules est un no-op pour le stub (le moteur décide en temps réel).
func (d *Driver) ApplyRules(rules []core.Rule) error { return nil }

// Start génère des connexions fictives et les soumet au moteur.
func (d *Driver) Start(ctx context.Context, engine *core.Engine) error {
	ctx, d.cancel = context.WithCancel(ctx)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Quelques destinations plausibles pour rendre la démo réaliste.
	targets := []struct {
		ip   string
		port int
		host string
	}{
		{"142.250.0.1", 443, "google.com"},
		{"140.82.0.1", 22, "github.com"},
		{"151.101.0.1", 443, "reddit.com"},
		{"192.0.2.66", 8080, "suspect.example"},
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			app := d.apps[rand.Intn(len(d.apps))]
			t := targets[rand.Intn(len(targets))]
			proto := core.ProtocolTCP
			if rand.Intn(3) == 0 {
				proto = core.ProtocolUDP
			}
			conn := core.Connection{
				ID:         fmt.Sprintf("c-%d", time.Now().UnixNano()),
				AppID:      app.ID,
				AppName:    app.Name,
				Protocol:   proto,
				LocalAddr:  "127.0.0.1",
				RemoteAddr: t.ip,
				RemotePort: t.port,
				Direction:  "out",
				At:         time.Now(),
			}
			_ = conn // Le moteur journalise via Decide ci-dessous.

			decision := engine.Decide(conn)
			_ = decision
			// Dans un vrai backend, on appliquerait la décision ici
			// (laisser passer / dropper le paquet) via l'API native.
		}
	}
}

// Stop arrête la boucle de simulation.
func (d *Driver) Stop() error {
	if d.cancel != nil {
		d.cancel()
	}
	return nil
}

// hostNames résout une IP en nom d'hôte (utilitaire, best-effort).
func hostNames(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}
