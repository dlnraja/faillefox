// Package threatintel agrège les indicateurs de compromission (IOC) depuis
// plusieurs sources publiques gratuites, et les met à disposition du
// corrélateur d'alertes.
//
// Sources intégrées (toutes gratuites, publiques, mises à jour par leurs
// communautés) :
//
//   - Abuse.ch MalwareBazaar  : échantillons de malwares + hashes
//   - Abuse.ch URLhaus        : URLs malveillantes
//   - Abuse.ch ThreatFox      : IOC (IP/domaines/hashes) attribués à APT
//   - AlienVault OTX          : pulses communautaires (IOC variés)
//   - StevenBlack/hosts       : domaines malveillants (déjà dans updater)
//
// Toutes ces sources sont CONSULTABLES sans clé API (Abuse.ch et StevenBlack
// totalement libres ; OTX propose une clé optionnelle pour un quota plus
// élevé).
//
// Le package ne fait QUE consommer des données publiques. Il ne génère pas
// de signatures maison (ce serait dangereux : faux positifs sans infra de
// test). Il met en cache les IOC pour que le corrélateur puisse les croiser.
package threatintel

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

// IOCType catégorise un indicateur de compromission.
type IOCType string

const (
	IOCIP       IOCType = "ip"       // adresse IP malveillante
	IOCDomain   IOCType = "domain"   // domaine malveillant
	IOCURL      IOCType = "url"      // URL malveillante
	IOCHash     IOCType = "hash"     // hash de fichier malveillant (SHA256)
)

// IOC est un indicateur de compromission issu d'une source publique.
type IOC struct {
	Value      string    `json:"value"`       // l'IOC lui-même (IP, domaine, hash...)
	Type       IOCType   `json:"type"`        // catégorie
	Source     string    `json:"source"`      // "abuse.ch", "otx", "misp"...
	Confidence int       `json:"confidence"`  // 1-10 (selon la source)
	Tags       []string  `json:"tags"`        // ex: ["apt29", "ransomware"]
	FirstSeen  time.Time `json:"first_seen"`
}

// Aggregator collecte les IOC de plusieurs sources et les indexe par valeur.
type Aggregator struct {
	mu     sync.RWMutex
	byVal  map[string][]IOC // clé: lowercase(value) -> sources qui l'ont vu
	client *http.Client
}

// New crée un agrégateur vide.
func New() *Aggregator {
	return &Aggregator{
		byVal:  make(map[string][]IOC),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchAll télécharge les IOC de toutes les sources configurées. Non bloquant
// sur une source qui échoue : on logge et on continue.
func (a *Aggregator) FetchAll(ctx context.Context) (int, error) {
	total := 0
	for name, fn := range a.sources() {
		n, err := fn(ctx, a)
		if err != nil {
			log.Printf("[threatintel] %s: %v (ignoré)", name, err)
			continue
		}
		log.Printf("[threatintel] %s: %d IOC", name, n)
		total += n
	}
	return total, nil
}

// sources retourne la map nom -> fonction de fetch.
// Chaque fonction peuple l'agrégateur via Add.
func (a *Aggregator) sources() map[string]func(context.Context, *Aggregator) (int, error) {
	return map[string]func(context.Context, *Aggregator) (int, error){
		"abuse.ch/ThreatFox": a.fetchThreatFox,
		"abuse.ch/URLhaus":   a.fetchURLhaus,
		"AlienVault-OTX":     a.fetchOTX,
	}
}

// Add ajoute un IOC à l'index (thread-safe).
func (a *Aggregator) Add(ioc IOC) {
	key := norm(ioc.Value)
	a.mu.Lock()
	a.byVal[key] = append(a.byVal[key], ioc)
	a.mu.Unlock()
}

// Lookup renvoie tous les IOC connus pour une valeur donnée (IP/domaine/hash).
// Plus il y a de sources qui voient le même IOC, plus la confiance est haute.
func (a *Aggregator) Lookup(value string) []IOC {
	key := norm(value)
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]IOC, len(a.byVal[key]))
	copy(out, a.byVal[key])
	return out
}

// Sources renvoie le nombre de sources distinctes qui voient une valeur.
// C'est le score de corrélation : 3 sources = forte confiance.
func (a *Aggregator) Sources(value string) int {
	iocs := a.Lookup(value)
	seen := map[string]struct{}{}
	for _, i := range iocs {
		seen[i.Source] = struct{}{}
	}
	return len(seen)
}

// Stats renvoie un résumé de l'index (pour l'UI).
func (a *Aggregator) Stats() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	stats := map[string]int{"total": len(a.byVal)}
	for _, iocs := range a.byVal {
		for _, i := range iocs {
			stats["by_source:"+i.Source]++
		}
	}
	return stats
}

// ---- sources spécifiques --------------------------------------------------

// fetchThreatFox télécharge les IOC d'Abuse.ch ThreatFox (JSON, sans clé API).
// Doc API : https://threatfox.abuse.ch/api/
func (a *Aggregator) fetchThreatFox(ctx context.Context, _ *Aggregator) (int, error) {
	url := "https://threatfox-api.abuse.ch/api/v1/?limit=1000"
	body, err := a.getJSON(ctx, url)
	if err != nil {
		return 0, err
	}
	// Format : {"query_status":"ok","data":[{ "ioc":"1.2.3.4","ioc_type":"ip:port",... }]}
	var resp struct {
		QueryStatus string `json:"query_status"`
		Data        []struct {
			IOC         string   `json:"ioc"`
			IOCType     string   `json:"ioc_type"`
			ThreatType  string   `json:"threat_type"`
			Tags        []string `json:"tags"`
			Confidence  int      `json:"confidence_level"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	n := 0
	for _, d := range resp.Data {
		iocType, val := parseThreatFoxType(d.IOCType, d.IOC)
		if val == "" {
			continue
		}
		a.Add(IOC{
			Value:      val,
			Type:       iocType,
			Source:     "abuse.ch/ThreatFox",
			Confidence: d.Confidence,
			Tags:       d.Tags,
			FirstSeen:  time.Now(),
		})
		n++
	}
	return n, nil
}

// fetchURLhaus télécharge les URLs malveillantes d'Abuse.ch URLhaus.
// Doc : https://urlhaus-api.abuse.ch/
func (a *Aggregator) fetchURLhaus(ctx context.Context, _ *Aggregator) (int, error) {
	url := "https://urlhaus-api.abuse.ch/v1/payloads/?limit=1000"
	body, err := a.getJSON(ctx, url)
	if err != nil {
		return 0, err
	}
	var resp struct {
		QueryStatus string `json:"query_status"`
		Payloads    []struct {
			URL    string `json:"url"`
			SHA256 string `json:"sha256_hash"`
		} `json:"payloads"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	n := 0
	for _, p := range resp.Payloads {
		if p.URL != "" {
			a.Add(IOC{Value: p.URL, Type: IOCURL, Source: "abuse.ch/URLhaus", Confidence: 7, FirstSeen: time.Now()})
		}
		if p.SHA256 != "" {
			a.Add(IOC{Value: strings.ToLower(p.SHA256), Type: IOCHash, Source: "abuse.ch/URLhaus", Confidence: 7, FirstSeen: time.Now()})
		}
		n++
	}
	return n, nil
}

// fetchOTX télécharge les pulses récents d'AlienVault OTX (sans clé, quota réduit).
// Doc : https://otx.alienvault.com/api/v1/indicators/...
func (a *Aggregator) fetchOTX(ctx context.Context, _ *Aggregator) (int, error) {
	// OTX expose des souscriptions publiques d'IOC récents.
	url := "https://otx.alienvault.com/api/v1/indicators/exports?type=IPv4&limit=1000"
	body, err := a.getJSON(ctx, url)
	if err != nil {
		return 0, err
	}
	// OTX renvoie du texte (une IP par ligne) pour les exports.
	lines := strings.Split(string(body), "\n")
	n := 0
	for _, line := range lines {
		ip := strings.TrimSpace(line)
		if ip == "" || strings.Contains(ip, "{") {
			continue
		}
		a.Add(IOC{Value: ip, Type: IOCIP, Source: "AlienVault-OTX", Confidence: 5, FirstSeen: time.Now()})
		n++
	}
	return n, nil
}

// ---- helpers --------------------------------------------------------------

// getJSON télécharge une URL et renvoie le body. User-Agent explicite.
func (a *Aggregator) getJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Faillefox/0.7 (+https://github.com/dlnraja/faillefox)")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return readAll(resp.Body)
}

// parseThreatFoxType mappe le type IOC de ThreatFox vers notre enum.
// Types possibles : "ip:port", "domain", "url", "md5_hash", "sha1_hash", "sha256_hash".
func parseThreatFoxType(tfType, value string) (IOCType, string) {
	switch tfType {
	case "ip:port":
		// "1.2.3.4:80" -> on garde juste l'IP pour le lookup
		if idx := strings.LastIndex(value, ":"); idx > 0 {
			return IOCIP, value[:idx]
		}
		return IOCIP, value
	case "domain":
		return IOCDomain, value
	case "url":
		return IOCURL, value
	case "sha256_hash":
		return IOCHash, strings.ToLower(value)
	case "md5_hash", "sha1_hash":
		return IOCHash, strings.ToLower(value)
	}
	return "", ""
}

// norm normalise une valeur d'IOC pour le lookup (minuscules, trim).
func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
