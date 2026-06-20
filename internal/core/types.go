// Package core contient le cœur du pare-feu Faillefox : le modèle de données,
// le moteur de règles, le journal et l'interface Driver que chaque backend
// natif (Windows WFP, Android VPNService, Linux nftables) doit implémenter.
//
// Le cœur ne sait RIEN du filtrage bas niveau. Il raisonne uniquement en
// termes d'applications, de règles et de décisions. C'est ce qui permet de
// partager toute la logique entre plateformes.
package core

import (
	"net"
	"time"
)

// Decision est le verdict appliqué à une connexion sortante.
type Decision string

const (
	DecisionAllow Decision = "allow" // Laisser passer
	DecisionDeny  Decision = "deny"  // Bloquer
	DecisionAsk   Decision = "ask"   // Demander à l'utilisateur (prompt)
)

// Protocol identifie le protocole réseau observé.
type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

// Connection décrit une connexion réseau sortante interceptée par un backend.
// C'est l'unité de base sur laquelle le moteur de règles prend une décision.
type Connection struct {
	ID         string   `json:"id"`
	AppID      string   `json:"app_id"`      // Identifiant de l'app (chemin exe, package Android, PID...)
	AppName    string   `json:"app_name"`    // Nom lisible
	Protocol   Protocol `json:"protocol"`
	LocalAddr  string   `json:"local_addr"`
	RemoteAddr string   `json:"remote_addr"`
	RemotePort int      `json:"remote_port"`
	Direction  string   `json:"direction"` // "out" en v1
	At         time.Time `json:"at"`
}

// HostName résout l'IP distante en nom d'hôte si possible (best-effort).
// Renvoie "" si la résolution échoue. Utilisé uniquement pour l'affichage.
func (c *Connection) HostName() string {
	if c.RemoteAddr == "" {
		return ""
	}
	ip := net.ParseIP(c.RemoteAddr)
	if ip == nil {
		return c.RemoteAddr
	}
	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

// Rule est une règle de filtrage. En v1 les règles sont principalement
// attachées à une application, mais le modèle prévoit les règles par
// port/IP pour le mode avancé.
type Rule struct {
	ID       string   `json:"id"`
	AppID    string   `json:"app_id"`
	Action   Decision `json:"action"`
	Protocol Protocol `json:"protocol"` // "" = tous
	Port     int      `json:"port"`      // 0 = tous
	IP       string   `json:"ip"`        // "" = toutes
	Note     string   `json:"note"`
	Created  time.Time `json:"created"`
}

// Match teste si la règle s'applique à la connexion donnée.
func (r Rule) Match(c *Connection) bool {
	if r.AppID != "" && r.AppID != c.AppID {
		return false
	}
	if r.Protocol != "" && r.Protocol != c.Protocol {
		return false
	}
	if r.Port != 0 && r.Port != c.RemotePort {
		return false
	}
	if r.IP != "" && r.IP != c.RemoteAddr {
		return false
	}
	return true
}

// Event est une entrée du journal : une décision prise par le moteur.
type Event struct {
	ID         string    `json:"id"`
	Connection Connection `json:"connection"`
	Decision   Decision   `json:"decision"`
	Reason     string     `json:"reason"` // "rule:abc123", "default", "ask"
	At         time.Time  `json:"at"`
}
