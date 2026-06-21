package core

import (
	"bufio"
	"strings"
	"sync"
)

// Blocklist est une liste de domaines/hosts à bloquer (façon Pi-hole local).
// Chargée depuis un fichier au format hosts (un domaine par ligne, ou
// "IP domaine" comme /etc/hosts). Les commentaires (#) et lignes vides
// sont ignorés.
//
// Utilisée par le moteur pour refuser les connexions vers des domaines
// connus comme trackers/publicitaires.
type Blocklist struct {
	mu      sync.RWMutex
	domains map[string]struct{}
}

// NewBlocklist crée une blocklist vide.
func NewBlocklist() *Blocklist {
	return &Blocklist{domains: make(map[string]struct{})}
}

// Add ajoute un domaine à la liste (normalisé en minuscules, sans port).
func (b *Blocklist) Add(domain string) {
	d := normalizeDomain(domain)
	if d == "" {
		return
	}
	b.mu.Lock()
	b.domains[d] = struct{}{}
	b.mu.Unlock()
}

// Contains teste si un domaine (ou l'un de ses parents) est dans la liste.
// Exemple : si "doubleclick.net" est bloqué, alors "ads.doubleclick.net"
// l'est aussi.
func (b *Blocklist) Contains(domain string) bool {
	d := normalizeDomain(domain)
	if d == "" {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Teste le domaine puis tous ses parents (sub.domain.tld -> domain.tld).
	for {
		if _, ok := b.domains[d]; ok {
			return true
		}
		idx := strings.Index(d, ".")
		if idx < 0 {
			return false
		}
		d = d[idx+1:]
		if !strings.Contains(d, ".") {
			return false
		}
	}
}

// Size renvoie le nombre d'entrées.
func (b *Blocklist) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.domains)
}

// LoadFromHosts parse un contenu au format hosts et peuple la liste.
// Format accepté (un domaine par ligne, whitespace collapsé) :
//
//	0.0.0.0 ads.example.com
//	tracker.example.org
//	# commentaire
func (b *Blocklist) LoadFromHosts(content string) int {
	n := 0
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		// "IP domaine" -> on prend le dernier champ ; "domaine" seul aussi.
		domain := fields[len(fields)-1]
		b.Add(domain)
		n++
	}
	return n
}

// normalizeDomain normalise un domaine pour le stockage/lookup.
func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimSuffix(d, ".") // FQDN racine
	return d
}
