// Package securitycenter est le chef d'orchestre de toutes les protections
// Faillefox. Il agrège l'état de chaque couche (pare-feu, DNS sinkhole,
// anti-malware, anti-pubs, anti-trackers, CVE, threat intel) et l'expose
// à l'UI via une vue synthétique.
//
// C'est l'équivalent du « Centre de sécurité Windows » ou de la console
// Bitdefender : un seul endroit où l'utilisateur voit si chaque protection
// est ACTIVE, INACTIVE ou LIMITÉE, et peut l'activer/désactiver.
package securitycenter

import (
	"sync"
	"time"
)

// Protection identifie une couche de protection.
type Protection string

const (
	ProtFirewall       Protection = "firewall"        // Pare-feu par application
	ProtDNS            Protection = "dns_sinkhole"    // DNS sinkhole (anti-pubs/trackers/malwares)
	ProtAntiAds        Protection = "anti_ads"        // Blocage publicitaires (via DNS)
	ProtAntiTrackers   Protection = "anti_trackers"   // Blocage trackers (via DNS)
	ProtAntiMalware    Protection = "anti_malware"    // Blocage domaines malwares (DNS + threat intel)
	ProtAntiAdware     Protection = "anti_adware"     // Blocage adware/PUP
	ProtAntiPhishing   Protection = "anti_phishing"   // Blocage phishing
	ProtAVScanner      Protection = "av_scanner"      // Scanner antivirus (ClamAV)
	ProtYARAScanner    Protection = "yara_scanner"    // Scanner YARA (signatures publiques)
	ProtCVEFeed        Protection = "cve_feed"        // Veille vulnérabilités (NVD)
	ProtThreatIntel    Protection = "threat_intel"    // Threat intel (Abuse.ch, OTX)
	ProtAutoUpdate     Protection = "auto_update"     // Auto-update des listes
	ProtFreshclam      Protection = "freshclam"       // MAJ signatures ClamAV
)

// Status est l'état d'une protection.
type Status string

const (
	StatusActive   Status = "active"   // fonctionne et protège
	StatusInactive Status = "inactive" // désactivée par l'utilisateur ou indispo
	StatusLimited  Status = "limited"  // fonctionne mais avec limites (ex: stub)
	StatusError    Status = "error"    // en erreur
)

// ProtectionState est l'état détaillé d'une protection (pour l'UI).
type ProtectionState struct {
	ID          Protection `json:"id"`
	Name        string     `json:"name"`         // nom lisible
	Category    string     `json:"category"`     // "Pare-feu", "Anti-pubs", etc.
	Status      Status     `json:"status"`       // active/inactive/limited/error
	Description string     `json:"description"`  // ce que fait cette protection
	Icon        string     `json:"icon"`         // emoji pour l'UI
	Stats       map[string]int `json:"stats,omitempty"` // compteurs (ex: domaines bloqués)
	LastEvent   *time.Time `json:"last_event,omitempty"`
}

// Center est le centre de sécurité. Thread-safe.
type Center struct {
	mu      sync.RWMutex
	states  map[Protection]*ProtectionState
}

// New crée un centre de sécurité avec toutes les protections connues,
// toutes en StatusInactive par défaut. Les modules les passent à Active
// quand ils démarrent.
func New() *Center {
	c := &Center{states: make(map[Protection]*ProtectionState)}
	for _, p := range allProtections() {
		// Statut par défaut explicite : sans ça, le champ Status vaut ""
		// (chaîne vide), qui n'est ni active ni inactive.
		p.Status = StatusInactive
		c.states[p.ID] = &p
	}
	return c
}

// allProtections retourne la liste complète des protections avec leurs
// métadonnées (nom, catégorie, description, icône).
func allProtections() []ProtectionState {
	return []ProtectionState{
		{ID: ProtFirewall, Name: "Pare-feu", Category: "Réseau",
			Icon: "🛡️", Description: "Filtre le trafic réseau par application (bloque/autorise chaque programme)"},
		{ID: ProtDNS, Name: "DNS sinkhole", Category: "DNS",
			Icon: "🌐", Description: "Résolveur DNS local qui bloque pubs/trackers/malwares pour tout le système"},
		{ID: ProtAntiAds, Name: "Anti-publicités", Category: "Anti-pubs",
			Icon: "🚫", Description: "Bloque les domaines publicitaires (Google Ads, doubleclick, etc.) via DNS"},
		{ID: ProtAntiTrackers, Name: "Anti-trackers", Category: "Vie privée",
			Icon: "🔍", Description: "Bloque les trackers (analytics, télémétrie, fingerprinting) via DNS"},
		{ID: ProtAntiMalware, Name: "Anti-malware (DNS)", Category: "Anti-malware",
			Icon: "🦠", Description: "Bloque les domaines malveillants connus (StevenBlack malware, Abuse.ch)"},
		{ID: ProtAntiAdware, Name: "Anti-adware / PUP", Category: "Anti-malware",
			Icon: "💢", Description: "Bloque les domaines adware et programmes potentiellement indésirables (PUP)"},
		{ID: ProtAntiPhishing, Name: "Anti-phishing", Category: "Anti-malware",
			Icon: "🎣", Description: "Bloque les domaines de phishing connus"},
		{ID: ProtAVScanner, Name: "Scanner antivirus (ClamAV)", Category: "Antivirus",
			Icon: "🔬", Description: "Scan de fichiers à la demande avec ClamAV (limité vs solutions commerciales)"},
		{ID: ProtYARAScanner, Name: "Scanner YARA", Category: "Antivirus",
			Icon: "🧬", Description: "Détection par signatures YARA publiques (YARA Forge, signature-base)"},
		{ID: ProtCVEFeed, Name: "Veille vulnérabilités (CVE)", Category: "Prévention",
			Icon: "📢", Description: "Surveille la base NVD et alerte si un logiciel installé a une faille connue"},
		{ID: ProtThreatIntel, Name: "Threat intelligence", Category: "Prévention",
			Icon: "🕵️", Description: "Agrège les IOC publics (Abuse.ch, OTX) et les croise pour prioriser les alertes"},
		{ID: ProtAutoUpdate, Name: "Auto-update des listes", Category: "Automatisation",
			Icon: "🔄", Description: "Rafraîchit automatiquement les listes DNS + CVE toutes les 6h"},
		{ID: ProtFreshclam, Name: "MAJ signatures ClamAV", Category: "Automatisation",
			Icon: "📥", Description: "Met à jour les signatures ClamAV via freshclam toutes les 2h"},
	}
}

// SetStatus met à jour le statut d'une protection.
func (c *Center) SetStatus(p Protection, s Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state, ok := c.states[p]; ok {
		state.Status = s
	}
}

// SetStats met à jour les statistiques d'une protection (ex: nb domaines bloqués).
func (c *Center) SetStats(p Protection, stats map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state, ok := c.states[p]; ok {
		state.Stats = stats
	}
}

// MarkEvent enregistre l'horodatage du dernier événement d'une protection.
func (c *Center) MarkEvent(p Protection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state, ok := c.states[p]; ok {
		now := time.Now()
		state.LastEvent = &now
	}
}

// States renvoie l'état de toutes les protections (pour l'API/UI).
func (c *Center) States() []ProtectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ProtectionState, 0, len(c.states))
	for _, s := range c.states {
		out = append(out, *s)
	}
	return out
}

// Summary renvoie un résumé haut niveau (combien d'actives, inactives...).
type Summary struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Limited  int `json:"limited"`
	Error    int `json:"error"`
	// Score global de protection (0-100) : pourcentage de protections actives.
	Score int `json:"score"`
}

// GetSummary calcule le résumé.
func (c *Center) GetSummary() Summary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := Summary{Total: len(c.states)}
	for _, state := range c.states {
		switch state.Status {
		case StatusActive:
			s.Active++
		case StatusInactive:
			s.Inactive++
		case StatusLimited:
			s.Limited++
		case StatusError:
			s.Error++
		}
	}
	if s.Total > 0 {
		// Active compte plein, Limited compte moitié.
		s.Score = (s.Active*100 + s.Limited*50) / s.Total
	}
	return s
}
