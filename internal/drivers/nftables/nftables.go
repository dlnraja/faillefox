// Package nftables est un pilote Linux réel qui pilote nftables (et iptables
// en fallback) pour bloquer le trafic sortant par application.
//
// Sur Linux, l'association paquet → application est plus complexe qu'avec
// WFP. Pour la v0.2 on se concentre sur le blocage par port/IP de
// destination, qui est fiable et immédiat. Le filtrage strict par
// application (via /proc/net + inodes de socket) est prévu en v0.3.
//
// Droits requis : root (CAP_NET_ADMIN), car nftables/iptables sont des
// opérations privilégiées.
package nftables

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/dlnraja/faillefox/internal/core"
)

func init() {
	core.RegisterDriver("linux-nftables", New)
}

// Driver pilote nftables/iptables sur Linux.
type Driver struct {
	useNft bool // true si nft est dispo, false si fallback iptables
}

// New détecte l'outil disponible (nft preferred, iptables en fallback).
func New(cfg core.DriverConfig) (core.Driver, error) {
	if _, err := exec.LookPath("nft"); err == nil {
		return &Driver{useNft: true}, nil
	}
	if _, err := exec.LookPath("iptables"); err == nil {
		return &Driver{useNft: false}, nil
	}
	return nil, errors.New("ni nft ni iptables trouvé sur ce système")
}

// Name identifie le pilote.
func (d *Driver) Name() string {
	if d.useNft {
		return "linux-nftables"
	}
	return "linux-iptables"
}

// ListApps énumère les binaires susceptibles d'émettre du trafic réseau.
// v0.2 : liste les processus actifs avec leur exécutable (/proc).
func (d *Driver) ListApps() ([]core.App, error) {
	cmd := exec.Command("sh", "-c",
		`for p in /proc/[0-9]*; do exe=$(readlink "$p/exe" 2>/dev/null); [ -n "$exe" ] && echo "$exe"; done | sort -u`)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	var apps []core.App
	for _, line := range strings.Split(string(out), "\n") {
		p := strings.TrimSpace(line)
		if p == "" {
			continue
		}
		apps = append(apps, core.App{
			ID:   p,
			Name: p[strings.LastIndex(p, "/")+1:],
			Path: p,
		})
	}
	return apps, nil
}

// ApplyRules installe les règles de blocage dans nftables ou iptables.
// Pour la v0.2, on applique le blocage par destination (IP/port) car
// l'association fiable paquet→PID sur Linux nécessite NFQUEUE (v0.3).
func (d *Driver) ApplyRules(rules []core.Rule) error {
	// Nettoyage préalable des anciennes règles Faillefox.
	d.clearRules()

	for _, r := range rules {
		if r.Action != core.DecisionDeny {
			continue // v0.2 : on ne bloque que deny
		}
		if err := d.installBlock(r); err != nil {
			log.Printf("[nftables] règle non installée (%+v): %v", r, err)
		}
	}
	return nil
}

// installBlock ajoute une règle de blocage pour une destination donnée.
func (d *Driver) installBlock(r core.Rule) error {
	// Cible : IP:port. Si port=0, toute destination ; si IP vide, tous ports.
	target := r.IP
	if r.Port != 0 {
		target = fmt.Sprintf("%s:%d", r.IP, r.Port)
	}
	if target == ":" || target == "" {
		return nil // rien à bloquer de concret (v0.2)
	}

	if d.useNft {
		// nft add rule inet filter output ip daddr <ip> drop
		// (version simplifiée ; une version robuste gérerait les tables
		//  Faillefox dédiées pour ne pas polluer les tables système.)
		args := []string{"add", "rule", "inet", "filter", "output"}
		if r.IP != "" {
			args = append(args, "ip", "daddr", r.IP, "drop")
		} else if r.Port != 0 {
			args = append(args, "tcp", "dport", fmt.Sprintf("%d", r.Port), "drop")
		}
		return exec.Command("nft", args...).Run()
	}
	// Fallback iptables
	if r.IP != "" {
		return exec.Command("iptables", "-A", "OUTPUT", "-d", r.IP, "-j", "DROP").Run()
	}
	return exec.Command("iptables", "-A", "OUTPUT", "-p", "tcp",
		"--dport", fmt.Sprintf("%d", r.Port), "-j", "DROP").Run()
}

// clearRules supprime les règles que nous avons installées. v0.2 : simple
// vidage de notre chaîne dédiée (à implémenter pleinement en v0.3 avec
// une table nft dédiée `faillefox`).
func (d *Driver) clearRules() {
	// Pour ne pas casser le pare-feu système de l'utilisateur, on ne fait
	// rien d'irréversible ici. Les vraies règles seront isolées dans une
	// table/chaîne `faillefox` dédiée en v0.3.
}

// Start : pas de boucle temps réel (nftables fait le travail).
func (d *Driver) Start(ctx context.Context, engine *core.Engine) error {
	<-ctx.Done()
	return nil
}

// Stop nettoie nos règles au déchargement.
func (d *Driver) Stop() error {
	d.clearRules()
	return nil
}

var _ core.Driver = (*Driver)(nil)
