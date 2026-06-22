// Package netfw est un pilote Windows réel qui pilote le Pare-feu Windows
// (Windows Firewall) via netsh advfirewall. Pour chaque application bloquée
// par l'utilisateur, il installe une règle de blocage sortant ciblée sur
// le chemin de l'exécutable.
//
// Ce pilote ne nécessite PAS de driver noyau : il s'appuie sur le Pare-feu
// Windows natif. Il requiert en revanche des droits administrateur pour
// exécuter netsh (élévation UAC ou exécution en tant que service).
//
// Limitations v0.2 : filtrage par application (block/allow sortant). Le
// filtrage par port/IP se fera via les filtres WFP avancés (v0.3).
package netfw

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/dlnraja/faillefox/internal/core"
)

func init() {
	core.RegisterDriver("windows-netfw", New)
}

// Driver implémente core.Driver pour le Pare-feu Windows (netsh).
type Driver struct {
	rulePrefix string // préfixe commun à toutes nos règles pour les retrouver
}

// New construit le pilote Windows.
func New(cfg core.DriverConfig) (core.Driver, error) {
	return &Driver{rulePrefix: "Faillefox-"}, nil
}

// Name identifie le pilote.
func (d *Driver) Name() string { return "windows-netfw" }

// ListApps énumère les processus en cours avec leur chemin d'exécutable.
// Utilise tasklist + wmic (disponibles sur tout Windows). C'est une approche
// simple ; une version future utilisera l'API IP Helper native.
func (d *Driver) ListApps() ([]core.App, error) {
	// wmci est déprécié sur les Windows très récents ; on tente d'abord
	// PowerShell Get-Process qui est fiable et multi-versions.
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Process | Where-Object { $_.Path } | Select-Object -ExpandProperty Path -Unique`)
	out, err := cmd.Output()
	if err != nil {
		// Fallback silencieux : si PowerShell n'est pas dispo, liste vide.
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
			Name: baseName(p),
			Path: p,
		})
	}
	return apps, nil
}

// ApplyRules (re)installe les règles dans le Pare-feu Windows.
// Stratégie : on supprime toutes les règles préfixées Faillefox-, puis on
// recrée celles correspondant aux règles de blocage/autorisation.
func (d *Driver) ApplyRules(rules []core.Rule) error {
	// 1. Nettoyage : supprimer toutes les anciennes règles Faillefox-.
	//    On ignore l'erreur (peut survenir si aucune règle n'existe).
	_ = exec.Command("netsh", "advfirewall", "firewall", "delete",
		"rule", "name="+d.rulePrefix+"*").Run()

	// 2. Recréation des règles par application.
	for _, r := range rules {
		if r.AppID == "" {
			continue // v0.2 : on ne gère que le filtrage par app
		}
		action := "block"
		if r.Action == core.DecisionAllow {
			action = "allow"
		}
		// Sécurité : on valide l'AppID (chemin d'exécutable) pour prévenir
		// l'injection de commande via netsh. On refuse les chemins contenant
		// des guillemets, des points-virgules, ou des caractères de shell.
		safeAppID := sanitizeNetshPath(r.AppID)
		if safeAppID == "" {
			log.Printf("[netfw] AppID suspect ignoré: %q", r.AppID)
			continue
		}
		name := d.rulePrefix + sanitize(safeAppID)
		args := []string{
			"advfirewall", "firewall", "add", "rule",
			"name=" + name,
			"dir=out",
			"action=" + action,
			"program=" + safeAppID,
			"enable=yes",
		}
		if err := exec.Command("netsh", args...).Run(); err != nil {
			log.Printf("[netfw] impossible d'appliquer la règle %s: %v", name, err)
		}
	}
	return nil
}

// Start : pour ce pilote, pas de boucle d'interception temps réel (le
// Pare-feu Windows fait le travail). On se contente de ne pas bloquer.
func (d *Driver) Start(ctx context.Context, engine *core.Engine) error {
	<-ctx.Done()
	return nil
}

// Stop nettoie les règles Faillefox- du Pare-feu Windows pour ne pas
// laisser de traces au déchargement.
func (d *Driver) Stop() error {
	return exec.Command("netsh", "advfirewall", "firewall", "delete",
		"rule", "name="+d.rulePrefix+"*").Run()
}

// sanitize produit un nom de règle valide (sans espaces/chemins).
func sanitize(s string) string {
	r := strings.NewReplacer(
		":", "", "\\", "_", "/", "_", " ", "_",
	)
	return r.Replace(s)
}

// sanitizeNetshPath valide un chemin d'exécutable avant de le passer à netsh.
// Refuse les caractères dangereux qui pourraient casser l'argument ou injecter
// une commande : guillemets, points-virgules, pipes, backticks, $, %, &.
// Renvoie "" si le chemin est suspect.
func sanitizeNetshPath(path string) string {
	if path == "" || len(path) > 1024 {
		return ""
	}
	// Caractères systématiquement refusés (injection shell Windows).
	dangerous := "\";|`$%&\x00!<>()\n\r"
	for _, c := range dangerous {
		if strings.ContainsRune(path, c) {
			return ""
		}
	}
	return path
}

// baseName extrait le nom de fichier d'un chemin.
func baseName(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if idx := strings.LastIndex(path, "."); idx > 0 {
		path = path[:idx]
	}
	return path
}

// Compile-time check : Driver implémente core.Driver.
var _ core.Driver = (*Driver)(nil)

// note: les commandes ici utilisent exec.Command pour interagir avec netsh.
// En cas d'échec (droits insuffisants), les erreurs sont loggées mais ne
// font pas tomber le démon — le pare-feu reste fonctionnel côté cœur.
var _ = fmt.Sprintf
