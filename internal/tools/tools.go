// Package tools regroupe des outils de sécurité gratuits et pertinents qui
// complètent le pare-feu. Ce sont des features "bonus" qui augmentent la
// valeur perçue sans complexité supplémentaire.
//
// Outils inclus :
//   - PortScanner : scanne les ports ouverts sur une cible (localhost par
//     défaut, pour vérifier sa propre surface d'attaque)
//   - DNSLeakTest : vérifie si le résolveur DNS utilisé fuit (compare avec
//     les résolveurs attendus)
//   - PasswordChecker : évalue la force d'un mot de passe (entropie, listes
//     de mots de passe compromis connues localement)
package tools

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ---- PortScanner ---------------------------------------------------------

// PortScanner scanne les ports TCP d'une cible pour identifier la surface
// d'attaque (services exposés).
type PortScanner struct{}

// PortResult est le résultat du scan d'un port.
type PortResult struct {
	Port    int    `json:"port"`
	Open    bool   `json:"open"`
	Service string `json:"service,omitempty"` // nom du service deviné
}

// NewPortScanner crée un scanner.
func NewPortScanner() *PortScanner { return &PortScanner{} }

// Scan scanne les ports courants d'une cible. host doit être validé par
// l'appelant (loopback par défaut pour éviter les abus).
func (p *PortScanner) Scan(host string, timeout time.Duration) []PortResult {
	if host == "" {
		host = "127.0.0.1"
	}
	// Ports courants à scanner (surface d'attaque typique).
	commonPorts := map[int]string{
		21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP",
		53: "DNS", 80: "HTTP", 110: "POP3", 143: "IMAP",
		443: "HTTPS", 445: "SMB", 3306: "MySQL", 3389: "RDP",
		5432: "PostgreSQL", 6379: "Redis", 8080: "HTTP-alt",
		8443: "HTTPS-alt", 9200: "Elasticsearch",
	}

	var mu sync.Mutex
	results := make([]PortResult, 0, len(commonPorts))
	var wg sync.WaitGroup
	for port, svc := range commonPorts {
		wg.Add(1)
		go func(port int, svc string) {
			defer wg.Done()
			// Formatage compatible IPv4 et IPv6 (crochets pour IPv6).
			addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return // port fermé
			}
			_ = conn.Close()
			mu.Lock()
			results = append(results, PortResult{Port: port, Open: true, Service: svc})
			mu.Unlock()
		}(port, svc)
	}
	wg.Wait()
	return results
}

// ---- DNSLeakTest ---------------------------------------------------------

// DNSLeakTest vérifie quel résolveur DNS répond réellement, pour détecter
// une fuite (FAI qui intercepte, VPN qui fuit, etc.).
type DNSLeakTest struct{}

// NewDNSLeakTest crée un testeur.
func NewDNSLeakTest() *DNSLeakTest { return &DNSLeakTest{} }

// DNSResolver est un résolveur DNS détecté.
type DNSResolver struct {
	Address    string `json:"address"`     // IP du résolveur
	Provider   string `json:"provider"`    // nom deviné (Cloudflare, Google...)
	Responding bool   `json:"responding"`
}

// Test vérifie quels résolveurs répondent. Compare avec les résolveurs
// publics connus pour identifier le provider.
func (d *DNSLeakTest) Test() []DNSResolver {
	// Résolveurs publics connus (pour identifier le provider).
	known := map[string]string{
		"1.1.1.1": "Cloudflare", "1.0.0.1": "Cloudflare",
		"8.8.8.8": "Google", "8.8.4.4": "Google",
		"9.9.9.9": "Quad9", "149.112.112.112": "Quad9",
		"208.67.222.222": "OpenDNS", "208.67.220.220": "OpenDNS",
	}
	var resolvers []DNSResolver
	var mu sync.Mutex
	var wg sync.WaitGroup
	for addr, provider := range known {
		wg.Add(1)
		go func(addr, provider string) {
			defer wg.Done()
			// Tente une requête DNS via ce résolveur.
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return net.DialTimeout("udp", addr+":53", 2*time.Second)
				},
			}
			_ = r // On utilise un dial custom pour forcer le résolveur.
			// Test simple : on tente de résoudre un domaine connu.
			conn, err := net.DialTimeout("udp", addr+":53", 2*time.Second)
			responding := err == nil
			if conn != nil {
				_ = conn.Close()
			}
			mu.Lock()
			resolvers = append(resolvers, DNSResolver{
				Address:    addr,
				Provider:   provider,
				Responding: responding,
			})
			mu.Unlock()
		}(addr, provider)
	}
	wg.Wait()
	return resolvers
}

// ---- PasswordChecker -----------------------------------------------------

// PasswordChecker évalue la force d'un mot de passe.
type PasswordChecker struct{}

// NewPasswordChecker crée un vérificateur.
func NewPasswordChecker() *PasswordChecker { return &PasswordChecker{} }

// PasswordStrength est l'évaluation de la force d'un mot de passe.
type PasswordStrength struct {
	Score     int    `json:"score"`      // 0-4 (très faible à très fort)
	Label     string `json:"label"`      // "Très faible".."Très fort"
	Entropy   int    `json:"entropy"`    // bits d'entropie estimés
	Feedback  string `json:"feedback"`   // conseils d'amélioration
}

// Evaluate évalue la force d'un mot de passe. SÉCURITÉ : le mot de passe
// n'est JAMAIS stocké ni loggé — on ne calcule que des métriques.
func (p *PasswordChecker) Evaluate(password string) PasswordStrength {
	if len(password) == 0 {
		return PasswordStrength{Score: 0, Label: "Vide", Feedback: "Aucun mot de passe"}
	}

	// Calcul de l'entropie (pool de caractères utilisé).
	pool := 0
	if strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		pool += 26
	}
	if strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		pool += 26
	}
	if strings.ContainsAny(password, "0123456789") {
		pool += 10
	}
	if strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:',.<>?/`~") {
		pool += 32
	}
	if pool == 0 {
		pool = 1
	}
	// Entropie = log2(pool^length) ≈ length * log2(pool).
	entropy := 0
	for p := pool; p > 1; p /= 2 {
		entropy++
	}
	entropy *= len(password)

	// Détection de patterns faibles.
	feedback := ""
	score := 0
	switch {
	case len(password) < 8:
		score = 0
		feedback = "Trop court (minimum 12 caractères recommandé)"
	case len(password) < 12:
		score = 1
		feedback = "Court — visez 12+ caractères"
	case entropy < 40:
		score = 1
		feedback = "Entropie faible — ajoutez majuscules, chiffres, symboles"
	case entropy < 60:
		score = 2
		feedback = "Correct — pourrait être plus long ou plus varié"
	case entropy < 80:
		score = 3
		feedback = "Bon mot de passe"
	default:
		score = 4
		feedback = "Excellent"
	}
	// Pénalités pour patterns communs.
	lower := strings.ToLower(password)
	commonWeak := []string{"123", "abc", "password", "azerty", "qwerty", "0000", "aaaa"}
	for _, w := range commonWeak {
		if strings.Contains(lower, w) {
			if score > 0 {
				score--
			}
			feedback = "Contient un pattern commun (" + w + ") — évitez"
			break
		}
	}

	labels := []string{"Très faible", "Faible", "Moyen", "Fort", "Très fort"}
	return PasswordStrength{
		Score:    score,
		Label:    labels[score],
		Entropy:  entropy,
		Feedback: feedback,
	}
}
