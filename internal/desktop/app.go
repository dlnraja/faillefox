// Package desktop est l'interface cliente graphique native de Faillefox.
//
// Contrairement au panneau web (navigateur) et au TUI (terminal), cette
// interface est une vraie FENÊTRE native de l'OS (barre de titre, boutons
// fermer/réduire, redimensionnement) construite avec Fyne (framework GUI
// standard de Go, cross-platform, rendu OpenGL natif).
//
// Lancement : `faillefox -gui` (connecté au démon tournant en arrière-plan).
//
// L'interface parle à l'API loopback du démon (même principe que le TUI et
// le panneau web), mais dans une vraie fenêtre desktop avec widgets natifs :
// boutons, onglets, tableaux, interrupteurs, barres de progression.
//
// Dépendance : fyne.io/fyne/v2 (nécessite CGO + OpenGL au build).
package desktop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Run lance l'application desktop. Bloquant jusqu'à fermeture de la fenêtre.
// port est le port du démon Faillefox (défaut 8443).
// token est le token d'authentification (requis pour les mutations).
func Run(port int, token string) error {
	a := app.New()
	a.SetIcon(nil) // icône par défaut (une icône custom sera ajoutée plus tard)
	w := a.NewWindow("Faillefox — Pare-feu")
	w.Resize(fyne.NewSize(900, 600))

	// Onglets : Dashboard, Règles, Sécurité, Outils.
	tabs := container.NewAppTabs(
		container.NewTabItem("🛡️ Dashboard", dashboardTab(port, token)),
		container.NewTabItem("📋 Règles", rulesTab(port, token)),
		container.NewTabItem("🔐 Sécurité", securityTab(port, token)),
		container.NewTabItem("🧰 Outils", toolsTab(port, token)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	w.SetContent(tabs)
	w.ShowAndRun()
	return nil
}

// client est le client HTTP parlant à l'API loopback.
type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func newClient(port int, token string) *client {
	return &client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		token:   token,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *client) get(path string) (map[string]any, error) {
	req, _ := http.NewRequest("GET", c.baseURL+path, nil)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
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

// dashboardTab construit l'onglet tableau de bord (statut + score).
func dashboardTab(port int, token string) fyne.CanvasObject {
	c := newClient(port, token)
	status := widget.NewLabel("Chargement…")
	score := widget.NewLabel("")

	refresh := func() {
		data, err := c.get("/api/status")
		if err != nil {
			status.SetText("⚠ Démon injoignable : " + err.Error())
			return
		}
		status.SetText(fmt.Sprintf("Pilote : %v | Règles : %v | Politique : %v",
			data["driver"], data["rules_count"], data["default_decision"]))

		sec, err := c.get("/api/security-center")
		if err == nil {
			if summary, ok := sec["summary"].(map[string]any); ok {
				score.SetText(fmt.Sprintf("Score de protection : %v%%\nProtections actives : %v / %v",
					summary["score"], summary["active"], summary["total"]))
			}
		}
	}

	refresh()
	btn := widget.NewButton("🔄 Actualiser", refresh)

	return container.NewVBox(
		widget.NewLabelWithStyle("🦊 Faillefox — Pare-feu libre",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		status,
		score,
		btn,
	)
}

// rulesTab construit l'onglet des règles.
func rulesTab(port int, token string) fyne.CanvasObject {
	c := newClient(port, token)
	list := widget.NewList(
		func() int { return 0 }, // sera dynamique
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)
	// Version simplifiée : on affiche le statut des règles.
	info := widget.NewLabel("Chargement des règles…")
	btn := widget.NewButton("🔄 Actualiser", func() {
		data, err := c.get("/api/rules")
		if err != nil {
			info.SetText("Erreur : " + err.Error())
			return
		}
		if rules, ok := data["rules"].([]any); ok {
			info.SetText(fmt.Sprintf("%d règle(s) chargée(s)", len(rules)))
		}
	})
	btn.OnTapped()
	return container.NewBorder(nil, btn, nil, nil, info)
}

// securityTab construit l'onglet centre de sécurité.
func securityTab(port int, token string) fyne.CanvasObject {
	c := newClient(port, token)
	info := widget.NewLabel("Chargement…")
	btn := widget.NewButton("🔄 Actualiser", func() {
		data, err := c.get("/api/security-center")
		if err != nil {
			info.SetText("Erreur : " + err.Error())
			return
		}
		if prots, ok := data["protections"].([]any); ok {
			text := ""
			for _, p := range prots {
				pm := p.(map[string]any)
				icon, _ := pm["icon"].(string)
				name, _ := pm["name"].(string)
				statusStr, _ := pm["status"].(string)
				text += fmt.Sprintf("%s  %-25s  %s\n", icon, name, statusStr)
			}
			info.SetText(text)
		}
	})
	btn.OnTapped()
	return container.NewBorder(nil, btn, nil, nil, container.NewVScroll(info))
}

// toolsTab construit l'onglet outils (ports + password).
func toolsTab(port int, token string) fyne.CanvasObject {
	c := newClient(port, token)
	result := widget.NewLabel("")

	scanBtn := widget.NewButton("🔌 Scanner mes ports", func() {
		data, err := c.get("/api/tools/ports")
		if err != nil {
			result.SetText("Erreur : " + err.Error())
			return
		}
		count := data["open_count"]
		if ports, ok := data["ports"].([]any); ok && len(ports) > 0 {
			text := fmt.Sprintf("%v port(s) ouvert(s) :\n", count)
			for _, p := range ports {
				pm := p.(map[string]any)
				text += fmt.Sprintf("  :%v %v\n", pm["port"], pm["service"])
			}
			result.SetText(text)
		} else {
			result.SetText("✓ Aucun port ouvert")
		}
	})

	genBtn := widget.NewButton("🎲 Générer un mot de passe", func() {
		data, err := c.get("/api/tools/gen-password?length=20")
		if err != nil {
			result.SetText("Erreur : " + err.Error())
			return
		}
		result.SetText(fmt.Sprintf("Mot de passe : %v", data["password"]))
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("🧰 Outils de sécurité",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		scanBtn,
		genBtn,
		result,
	)
}
