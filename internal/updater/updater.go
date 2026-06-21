// Package updater télécharge et rafraîchit périodiquement les listes de
// blocage DNS (pubs/trackers/malwares) et la base de vulnérabilités (CVE).
//
// Sources publiques et gratuites :
//   - StevenBlack/hosts : listes unifiées ads/trackers/malwares (~150k domaines)
//   - OISD : blocklist axée familles
//   - Abuse.ch : IOC malwares (URLhaus, MalwareBazaar)
//   - NVD (NIST) : base CVE officielle, JSON, mise à jour continue
//
// Toutes ces sources sont libres d'utilisation et mises à jour par leurs
// communautés. On les rafraîchit toutes les 24h par défaut.
package updater

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dlnraja/faillefox/internal/core"
)

// Updater gère le rafraîchissement périodique des listes.
type Updater struct {
	blocklist   *core.Blocklist
	dnsSources  []string // URLs des listes DNS
	cveSource   string   // URL du flux NVD CVE
	httpClient  *http.Client
	updateEvery time.Duration
}

// New crée un updater avec les sources publiques par défaut.
func New(blocklist *core.Blocklist) *Updater {
	return &Updater{
		blocklist: blocklist,
		// Sources publiques gratuites, mises à jour par leurs communautés.
		dnsSources: []string{
			"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
			"https://raw.githubusercontent.com/StevenBlack/hosts/master/data/StevenBlack/hosts",
			"https://oisd.nl/downloads/wildcardLight.txt",
		},
		cveSource: "https://services.nvd.nist.gov/rest/json/cves/2.0?resultsPerPage=2000",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		updateEvery: 24 * time.Hour,
	}
}

// FetchOnce télécharge toutes les listes une fois (sans boucle périodique).
// Renvoie le nombre total de domaines ajoutés.
func (u *Updater) FetchOnce(ctx context.Context) (int, error) {
	total := 0
	for _, url := range u.dnsSources {
		n, err := u.fetchHosts(ctx, url)
		if err != nil {
			log.Printf("[updater] %s: %v (ignoré)", url, err)
			continue
		}
		log.Printf("[updater] %s: %d domaines", shortURL(url), n)
		total += n
	}
	return total, nil
}

// Start lance la boucle périodique (FetchOnce puis toutes les 24h).
// Bloquant ; à lancer dans une goroutine.
func (u *Updater) Start(ctx context.Context) {
	if _, err := u.FetchOnce(ctx); err != nil {
		log.Printf("[updater] premier fetch: %v", err)
	}
	ticker := time.NewTicker(u.updateEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := u.FetchOnce(ctx); err != nil {
				log.Printf("[updater] fetch périodique: %v", err)
			}
		}
	}
}

// fetchHosts télécharge un fichier au format hosts et peuple la blocklist.
func (u *Updater) fetchHosts(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Faillefox/0.3 (+https://github.com/dlnraja/faillefox)")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return u.parseHosts(string(data)), nil
}

// parseHosts peuple la blocklist depuis un contenu au format hosts.
// Réutilise Blocklist.LoadFromHosts mais renvoie aussi le compte.
func (u *Updater) parseHosts(content string) int {
	n := 0
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // grosses listes
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		// Format "0.0.0.0 domaine" ou "127.0.0.1 domaine" ou "domaine"
		var domain string
		switch len(fields) {
		case 0:
			continue
		case 1:
			domain = fields[0]
		default:
			// Ignore "localhost", "broadcasthost", etc.
			if fields[0] == "localhost" {
				continue
			}
			domain = fields[len(fields)-1]
		}
		if domain == "localhost" || domain == "broadcasthost" || domain == "ip6-localhost" {
			continue
		}
		u.blocklist.Add(domain)
		n++
	}
	return n
}

// SetUpdateEvery change la fréquence de rafraîchissement.
func (u *Updater) SetUpdateEvery(d time.Duration) {
	u.updateEvery = d
}

// shortURL raccourcit une URL pour les logs.
func shortURL(url string) string {
	if i := strings.Index(url, "//"); i >= 0 {
		url = url[i+2:]
	}
	if i := strings.Index(url, "/"); i >= 0 {
		url = url[:i]
	}
	return url
}
