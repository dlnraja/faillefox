// Package correlate croise plusieurs sources (threat intel, CVE, activité
// locale observée) pour PRIORISER les alertes.
//
// Principe : une alerte simple "IP suspecte" est utile, mais une alerte
// "IP vue par 3 sources distinctes (Abuse.ch + OTX + MISP) ET émise par un
// logiciel qui a une CVE ouverte" est CRITIQUE. Le corrélateur transforme
// des signaux faibles en alertes priorisées.
//
// Le score de corrélation combine :
//   - nombre de sources threat intel qui voient l'IOC (+30 par source)
//   - présence d'une CVE sur le logiciel émetteur (+40)
//   - profil réseau (réseau public = +20)
//   - répétition (déjà vu N fois = +N*5)
package correlate

import (
	"github.com/dlnraja/faillefox/internal/core"
	"github.com/dlnraja/faillefox/internal/threatintel"
)

// Severity est le niveau de criticité d'une alerte corrélée.
type Severity string

const (
	SeverityInfo     Severity = "info"     // score < 30
	SeverityLow      Severity = "low"      // 30-59
	SeverityMedium   Severity = "medium"   // 60-89
	SeverityHigh     Severity = "high"     // 90-119
	SeverityCritical Severity = "critical" // >= 120
)

// Alert est une alerte corrélée (sortie du corrélateur).
type Alert struct {
	Title      string          `json:"title"`
	Severity   Severity        `json:"severity"`
	Score      int             `json:"score"`
	Connection core.Connection `json:"connection,omitempty"`
	Reasons    []string        `json:"reasons"` // pourquoi ce score
	IOC        []threatintel.IOC `json:"ioc,omitempty"`
}

// Correlator combine les sources pour produire des alertes priorisées.
type Correlator struct {
	threat *threatintel.Aggregator
	// Map des CVE par logiciel (clé: lowercase(nom)). Vide si pas de veille CVE.
	cvesBySoftware map[string]bool
	// Profil réseau courant (affecte le score).
	profile core.Profile
}

// New crée un corrélateur lié à un agrégateur threat intel.
func New(threat *threatintel.Aggregator) *Correlator {
	return &Correlator{
		threat:          threat,
		cvesBySoftware:  make(map[string]bool),
		profile:         core.ProfileHome,
	}
}

// SetCVEIndex enregistre l'index des logiciels ayant au moins une CVE connue.
// Clé = lowercase(nom du logiciel).
func (c *Correlator) SetCVEIndex(softwareWithCVE map[string]bool) {
	c.cvesBySoftware = softwareWithCVE
}

// SetProfile change le profil réseau courant (affecte le seuil).
func (c *Correlator) SetProfile(p core.Profile) {
	c.profile = p
}

// EvaluateConnection analyse une connexion observée et produit une alerte
// si elle mérite attention (sinon nil). C'est le cœur du corrélateur.
func (c *Correlator) EvaluateConnection(conn core.Connection) *Alert {
	score := 0
	var reasons []string

	// 1. Threat intel : l'IP distante est-elle connue comme malveillante ?
	iocs := c.threat.Lookup(conn.RemoteAddr)
	nSources := c.threat.Sources(conn.RemoteAddr)
	if nSources > 0 {
		points := nSources * 30
		score += points
		reasons = append(reasons, "IP dans "+itoa(nSources)+" source(s) threat intel")
	}

	// 2. CVE sur le logiciel émetteur ?
	appLower := lowerAppName(conn.AppName)
	if has, ok := c.cvesBySoftware[appLower]; ok && has {
		score += 40
		reasons = append(reasons, "logiciel émetteur a une CVE connue")
	}

	// 3. Profil réseau : sur un réseau public, on est plus strict.
	if c.profile == core.ProfilePublic {
		score += 20
		reasons = append(reasons, "réseau public")
	}

	// 4. Décision du moteur : si deny/ask, on booste.
	// (géré par l'appelant ; ici on regarde juste les signaux externes)

	// Si pas de signal significatif, pas d'alerte.
	if score < 30 {
		return nil
	}

	return &Alert{
		Title:      "Connexion suspecte vers " + conn.RemoteAddr,
		Severity:   severityFromScore(score),
		Score:      score,
		Connection: conn,
		Reasons:    reasons,
		IOC:        iocs,
	}
}

// severityFromScore mappe un score à une sévérité.
func severityFromScore(score int) Severity {
	switch {
	case score >= 120:
		return SeverityCritical
	case score >= 90:
		return SeverityHigh
	case score >= 60:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// lowerAppName normalise un nom d'app pour le lookup CVE.
// "Google Chrome" -> "chrome", "/usr/bin/curl" -> "curl".
func lowerAppName(name string) string {
	// On prend le dernier composant du chemin (séparateur / ou \).
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	// lowercase + retire l'extension.
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if ch == '.' {
			break // extension
		}
		out = append(out, ch)
	}
	return string(out)
}

// itoa convertit un int en string (évite d'importer strconv pour 1 usage).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
