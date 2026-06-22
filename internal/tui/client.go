// Package tui est une interface client riche pour terminal (TUI), alternative
// au panneau web. Elle parle à l'API loopback du démon Faillefox et affiche
// en plein écran :
//
//   - tableau de bord (centre de sécurité : score, protections actives)
//   - liste des règles
//   - journal temps réel
//   - paramètres
//
// Lancement : `faillefox -tui` (connecté au démon tournant en arrière-plan
// sur 127.0.0.1:8443, ou à un autre port via -port).
//
// L'interface est volontairement simple (rendu ANSI manuel) pour zéro
// dépendance externe et une compatibilité maximale (Windows console,
// Terminal macOS/Linux, SSH).
package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client est le client TUI qui parle à l'API Faillefox.
type Client struct {
	baseURL string
	http    *http.Client
}

// New crée un client connecté au démon sur 127.0.0.1:port.
func New(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Run lance l'interface TUI. Bloquant jusqu'à ce que l'utilisateur quitte.
func (c *Client) Run() error {
	clearScreen()
	for {
		c.renderDashboard()
		fmt.Println()
		fmt.Println("  Actions: [r] Règles  [l] Journal  [s] Sécurité  [p] Paramètres  [q] Quitter")
		fmt.Print("  > ")
		var key string
		_, _ = fmt.Scanln(&key)
		switch strings.ToLower(key) {
		case "q", "quit", "exit":
			fmt.Println("  Au revoir 👋")
			return nil
		case "r":
			c.viewRules()
		case "l":
			c.viewLogs()
		case "s":
			c.viewSecurity()
		case "p":
			c.viewSettings()
		case "":
			continue
		}
	}
}

// renderDashboard affiche le tableau de bord principal.
func (c *Client) renderDashboard() {
	clearScreen()
	fmt.Println(bold + "  ╔════════════════════════════════════════════════════╗")
	fmt.Println("  ║           🦊  FAILLEFOX  —  Pare-feu libre          ║")
	fmt.Println("  ╚════════════════════════════════════════════════════╝" + reset)
	fmt.Println()

	// État du démon.
	status, err := c.fetchStatus()
	if err != nil {
		fmt.Println("  " + red + "⚠ Démon injoignable : " + err.Error() + reset)
		fmt.Println("  Lancez d'abord : faillefox (sans -tui)")
		return
	}
	fmt.Printf("  Pilote : %v   |   Règles : %v   |   Politique : %v\n\n",
		status["driver"], status["rules_count"], status["default_decision"])

	// Centre de sécurité.
	sec, err := c.fetchSecurity()
	if err == nil {
		if summary, ok := sec["summary"].(map[string]any); ok {
			score := int(summary["score"].(float64))
			color := green
			if score < 40 {
				color = red
			} else if score < 75 {
				color = yellow
			}
			fmt.Printf("  Score de protection : %s%d%%%s\n", color, score, reset)
			fmt.Printf("  Protections : %d actives / %d total\n",
				int(summary["active"].(float64)), int(summary["total"].(float64)))
		}
	}
	fmt.Println()
}

// viewRules affiche les règles courantes.
func (c *Client) viewRules() {
	clearScreen()
	fmt.Println(bold + "  📋 RÈGLES" + reset)
	fmt.Println("  ─────────────────────────────────────────────────")
	rules, err := c.fetchSlice("/api/rules")
	if err != nil {
		fmt.Println("  " + red + err.Error() + reset)
	} else if len(rules) == 0 {
		fmt.Println("  Aucune règle définie.")
	} else {
		for _, r := range rules {
			app, _ := r["app_id"].(string)
			action, _ := r["action"].(string)
			port := r["port"]
			col := green
			if action == "deny" {
				col = red
			}
			fmt.Printf("  %s%-6s%s  %v  port=%v\n", col, action, reset, app, port)
		}
	}
	fmt.Println("\n  [Entrée] Retour")
	var skip string
	_, _ = fmt.Scanln(&skip)
}

// viewLogs affiche les derniers événements du journal.
func (c *Client) viewLogs() {
	clearScreen()
	fmt.Println(bold + "  📜 JOURNAL (derniers événements)" + reset)
	fmt.Println("  ─────────────────────────────────────────────────")
	events, err := c.fetchSlice("/api/events")
	if err != nil {
		// /api/events est en SSE, on ne peut pas le fetcher simplement.
		// On affiche un message et on propose le statut à la place.
		fmt.Println("  (Le journal temps réel est disponible dans le panneau web)")
		fmt.Println("  Ouvrez http://127.0.0.1:8443 dans votre navigateur.")
	} else {
		for _, e := range events {
			conn, _ := e["Connection"].(map[string]any)
			dec, _ := e["Decision"].(string)
			col := green
			if dec == "deny" {
				col = red
			}
			app, _ := conn["app_name"].(string)
			addr, _ := conn["remote_addr"].(string)
			fmt.Printf("  %s%-6s%s  %s → %s\n", col, dec, reset, app, addr)
		}
	}
	fmt.Println("\n  [Entrée] Retour")
	var skip string
	_, _ = fmt.Scanln(&skip)
}

// viewSecurity affiche le centre de sécurité en détail.
func (c *Client) viewSecurity() {
	clearScreen()
	fmt.Println(bold + "  🛡️  CENTRE DE SÉCURITÉ" + reset)
	fmt.Println("  ─────────────────────────────────────────────────")
	sec, err := c.fetchSecurity()
	if err != nil {
		fmt.Println("  " + red + err.Error() + reset)
	} else {
		prots, _ := sec["protections"].([]any)
		for _, p := range prots {
			pm, _ := p.(map[string]any)
			icon, _ := pm["icon"].(string)
			name, _ := pm["name"].(string)
			statusStr, _ := pm["status"].(string)
			col := green
			switch statusStr {
			case "inactive":
				col = faint
			case "limited":
				col = yellow
			case "error":
				col = red
			}
			fmt.Printf("  %s  %-30s %s%s%s\n", icon, name, col, statusStr, reset)
		}
	}
	fmt.Println("\n  [Entrée] Retour")
	var skip string
	_, _ = fmt.Scanln(&skip)
}

// viewSettings affiche les paramètres.
func (c *Client) viewSettings() {
	clearScreen()
	fmt.Println(bold + "  ⚙️  PARAMÈTRES" + reset)
	fmt.Println("  ─────────────────────────────────────────────────")
	s, err := c.fetchMap("/api/settings")
	if err != nil {
		fmt.Println("  " + red + err.Error() + reset)
	} else {
		keys := []string{"ui_mode", "theme", "profile", "notifications",
			"firewall", "dns_sinkhole", "anti_ads", "anti_malware",
			"anti_ransomware", "av_scanner", "cve_feed", "threat_intel",
			"gamification", "auto_update"}
		labels := map[string]string{
			"ui_mode": "Mode interface", "theme": "Thème", "profile": "Profil réseau",
			"notifications": "Notifications", "firewall": "Pare-feu", "dns_sinkhole": "DNS sinkhole",
			"anti_ads": "Anti-pubs", "anti_malware": "Anti-malware", "anti_ransomware": "Anti-ransomware",
			"av_scanner": "Scanner ClamAV", "cve_feed": "Veille CVE", "threat_intel": "Threat intel",
			"gamification": "Gamification", "auto_update": "Auto-update",
		}
		for _, k := range keys {
			v := s[k]
			col := green
			if b, ok := v.(bool); ok && !b {
				col = faint
			}
			fmt.Printf("  %-20s %s%v%s\n", labels[k]+":", col, v, reset)
		}
	}
	fmt.Println("\n  (Modifications via panneau web ou API /api/settings)")
	fmt.Println("  [Entrée] Retour")
	var skip string
	_, _ = fmt.Scanln(&skip)
}

// ---- helpers HTTP --------------------------------------------------------

func (c *Client) fetchStatus() (map[string]any, error) {
	return c.fetchMap("/api/status")
}

func (c *Client) fetchSecurity() (map[string]any, error) {
	return c.fetchMap("/api/security-center")
}

func (c *Client) fetchMap(path string) (map[string]any, error) {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Client) fetchSlice(path string) ([]map[string]any, error) {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var s []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		// Peut être un objet (ex: events SSE), pas un tableau.
		return nil, err
	}
	return s, nil
}

// ---- codes couleur ANSI --------------------------------------------------

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	faint  = "\033[2m"
)

// clearScreen efface le terminal (compatible Windows 10+/Unix).
func clearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

// Compile-time check : on expose bien io pour usage futur (export logs).
var _ = io.Discard
var _ = os.Stdin
