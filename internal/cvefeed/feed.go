// Package cvefeed consulte la base publique NVD (National Vulnerability
// Database du NIST) pour détecter les CVE affectant les logiciels installés
// localement, et émet des alertes.
//
// La base NVD est GRATUITE et PUBLIQUE — c'est la source officielle des CVE
// aux États-Unis, utilisée mondialement. On la consomme via son API JSON
// officielle (https://services.nvd.nist.gov/rest/json/cves/2.0).
//
// IMPORTANT : ce module NE détecte que les vulnérabilités CONNUES et
// INVENTORIÉES dans la NVD. Il ne fait pas d'analyse heuristique ni de
// sandboxing — ce n'est pas un AV. Il complète le bouclier réseau/DNS en
// signalant les logiciels à mettre à jour.
package cvefeed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Alert signale qu'un logiciel installé est affecté par une CVE.
type Alert struct {
	Software    string `json:"software"`     // ex: "curl 7.74.0"
	CVE         string `json:"cve"`          // ex: "CVE-2023-38545"
	Severity    string `json:"severity"`     // LOW/MEDIUM/HIGH/CRITICAL
	Description string `json:"description"`  // résumé de la CVE
	URL         string `json:"url"`          // lien NVD
}

// Feed interroge la NVD et compare avec les logiciels installés.
type Feed struct {
	httpClient *http.Client
	// index des CVE par nom de logiciel (construit au démarrage).
	mu   sync.Mutex
	idx  map[string][]cveEntry // clé: lowercase(product)
}

type cveEntry struct {
	cve         string
	severity    string
	description string
	url         string
}

// New crée un feed CVE prêt à interroger la NVD.
func New() *Feed {
	return &Feed{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		idx:        make(map[string][]cveEntry),
	}
}

// RefreshAll télécharge les CVE récentes (dernier mois) et peuple l'index.
// Limite volontairement à resultsPerPage pour ne pas saturer l'API NVD
// (rate limit public : 5 req/30s sans clé API).
func (f *Feed) RefreshAll(ctx context.Context) error {
	// Fenêtre temporelle : 30 derniers jours.
	start := time.Now().AddDate(0, 0, -30).Format("2006-01-02T00:00:00.000")
	url := fmt.Sprintf(
		"https://services.nvd.nist.gov/rest/json/cves/2.0?resultsPerPage=2000&pubStartDate=%s",
		start,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Faillefox/0.3 CVE-feed")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NVD HTTP %d", resp.StatusCode)
	}

	var apiResp nvdResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	f.mu.Lock()
	f.idx = make(map[string][]cveEntry)
	for _, vuln := range apiResp.Vulnerabilities {
		cve := vuln.CVE
		entry := cveEntry{
			cve:  cve.ID,
			url:  "https://nvd.nist.gov/vuln/detail/" + cve.ID,
		}
		// Sévérité + description (CVSS v3 si dispo).
		if len(cve.Metrics.CVSSMetricV31) > 0 {
			entry.severity = cve.Metrics.CVSSMetricV31[0].CVSSData.BaseSeverity
			entry.description = cve.Metrics.CVSSMetricV31[0].CVSSData.Description
		}
		if entry.description == "" && len(cve.Descriptions) > 0 {
			entry.description = cve.Descriptions[0].Value
		}
		// Indexation par produit (CPE).
		for _, conf := range cve.Configurations {
			for _, node := range conf.Nodes {
				for _, m := range node.CPEMatch {
					// CPE format: cpe:2.3:a:vendor:product:version:...
					product := extractCPEField(m.Criteria, "product")
					if product != "" {
						f.idx[strings.ToLower(product)] = append(
							f.idx[strings.ToLower(product)], entry)
					}
				}
			}
		}
	}
	f.mu.Unlock()

	log.Printf("[cve] index construit: %d produits surveillés", len(f.idx))
	return nil
}

// CheckSoftware compare une liste de logiciels installés (nom + version)
// à l'index des CVE et renvoie les alertes.
//
// La correspondance est approchée (par nom de produit) ; on ne vérifie pas
// la version exacte dans cette v0.3 — on signale qu'une CVE existe pour ce
// produit, à l'utilisateur de vérifier la version concernée.
func (f *Feed) CheckSoftware(installed []Software) []Alert {
	f.mu.Lock()
	defer f.mu.Unlock()
	var alerts []Alert
	for _, sw := range installed {
		key := strings.ToLower(sw.Name)
		if entries, ok := f.idx[key]; ok {
			// On évite les doublons : au max 3 CVE par logiciel.
			for i, e := range entries {
				if i >= 3 {
					break
				}
				alerts = append(alerts, Alert{
					Software:    fmt.Sprintf("%s %s", sw.Name, sw.Version),
					CVE:         e.cve,
					Severity:    e.severity,
					Description: truncate(e.description, 200),
					URL:         e.url,
				})
			}
		}
	}
	return alerts
}

// Software est un logiciel installé localement (nom + version).
type Software struct {
	Name    string
	Version string
}

// ---- Structures JSON de l'API NVD 2.0 ------------------------------------

type nvdResponse struct {
	TotalResults  int           `json:"totalResults"`
	Vulnerabilities []nvdVuln   `json:"vulnerabilities"`
}

type nvdVuln struct {
	CVE nvdCVE `json:"cve"`
}

type nvdCVE struct {
	ID            string         `json:"id"`
	Descriptions  []nvdDesc      `json:"descriptions"`
	Metrics       nvdMetrics     `json:"metrics"`
	Configurations []nvdConfig   `json:"configurations"`
}

type nvdDesc struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type nvdMetrics struct {
	CVSSMetricV31 []nvdCVSS `json:"cvssMetricV31"`
}

type nvdCVSS struct {
	CVSSData nvdCVSSData `json:"cvssData"`
}

type nvdCVSSData struct {
	BaseSeverity string `json:"baseSeverity"`
	Description  string `json:"description"`
}

type nvdConfig struct {
	Nodes []nvdNode `json:"nodes"`
}

type nvdNode struct {
	CPEMatch []nvdCPE `json:"cpeMatch"`
}

type nvdCPE struct {
	Criteria string `json:"criteria"`
}

// extractCPEField récupère un champ d'un CPE 2.3 formaté.
// CPE: cpe:2.3:a:vendor:product:version:...
func extractCPEField(cpe, field string) string {
	parts := strings.Split(cpe, ":")
	// parts[0]="cpe", [1]="2.3", [2]="a", [3]=vendor, [4]=product, [5]=version...
	if len(parts) < 5 {
		return ""
	}
	switch field {
	case "vendor":
		return parts[3]
	case "product":
		return parts[4]
	case "version":
		if len(parts) > 5 {
			return parts[5]
		}
	}
	return ""
}

// truncate coupe une chaîne à max caractères avec ellipse.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
