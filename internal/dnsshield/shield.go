// Package dnsshield implémente un résolveur DNS local qui bloque les
// domaines présents dans des listes publiques (pubs, trackers, malwares).
//
// Principe (façon Pi-hole / AdGuard) :
//   1. On démarre un serveur DNS sur 127.0.0.1:5353 (port non privilégié).
//   2. Pour chaque requête :
//      a. Si le domaine est dans une blocklist -> on répond 0.0.0.0 (sinkhole).
//      b. Sinon, on forward vers un upstream public (1.1.1.1, 9.9.9.9).
//   3. Pour activer le blocage au niveau système, on configure le système
//      pour utiliser 127.0.0.1 comme résolveur.
//
// Cela bloque les pubs/trackers/malwares POUR TOUTES les apps de la machine,
// pas seulement le navigateur — y compris les apps mobiles via le VPN.
//
// Dépendance externe : github.com/miekg/dns (pure Go, cross-platform).
package dnsshield

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/dlnraja/faillefox/internal/core"
)

// Shield est le résolveur DNS local avec sinkhole.
type Shield struct {
	mu        sync.RWMutex
	blocklist *core.Blocklist
	upstreams []string // serveurs DNS de fallback (1.1.1.1, 9.9.9.9)
	server    *dns.Server
	port      int
}

// New crée un résolveur DNS sur le port donné (loopback uniquement).
func New(port int) *Shield {
	return &Shield{
		port:      port,
		blocklist: core.NewBlocklist(),
		// Upstreams publics, gratuits, respectueux de la vie privée.
		// 1.1.1.1 = Cloudflare (promesse no-log), 9.9.9.9 = Quad9 (filtre malwares).
		upstreams: []string{"1.1.1.1:53", "9.9.9.9:53", "8.8.8.8:53"},
	}
}

// SetBlocklist remplace la liste des domaines bloqués.
func (s *Shield) SetBlocklist(bl *core.Blocklist) {
	s.mu.Lock()
	s.blocklist = bl
	s.mu.Unlock()
}

// Start démarre le serveur DNS. Bloquant. Le serveur n'écoute que sur
// 127.0.0.1 (loopback), jamais exposé.
func (s *Shield) Start(ctx context.Context) error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleDNS)

	s.server = &dns.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Net:     "udp",
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("dns: %w", err)
	case <-time.After(200 * time.Millisecond):
		// Le serveur a démarré (ListenAndServe est bloquant).
		log.Printf("[dns] résolveur sinkhole sur 127.0.0.1:%d", s.port)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop ferme le serveur DNS.
func (s *Shield) Stop() error {
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

// handleDNS traite une requête DNS : sinkhole si bloqué, sinon forward.
func (s *Shield) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	domain := canonicalDomain(r.Question[0].Name)

	s.mu.RLock()
	blocked := s.blocklist.Contains(domain)
	s.mu.RUnlock()

	if blocked {
		// Sinkhole : on répond 0.0.0.0 (IPv4) ou :: (IPv6). Le client
		// croit que le domaine n'existe pas -> connexion impossible.
		s.respondSinkhole(w, r)
		return
	}

	// Forward vers l'upstream.
	resp, err := s.forward(r)
	if err != nil || resp == nil {
		// En cas d'erreur, on répond NXDOMAIN plutôt que de planter.
		s.respondFail(w, r)
		return
	}
	_ = w.WriteMsg(resp)
}

// forward relaie la requête vers les upstreams configurés (round-robin
// avec fallback sur le premier qui répond).
func (s *Shield) forward(r *dns.Msg) (*dns.Msg, error) {
	c := &dns.Client{Timeout: 3 * time.Second}
	s.mu.RLock()
	upstreams := s.upstreams
	s.mu.RUnlock()

	var lastErr error
	for _, up := range upstreams {
		resp, _, err := c.Exchange(r, up)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// respondSinkhole répond 0.0.0.0 (le domaine est "bloqué").
func (s *Shield) respondSinkhole(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeAAAA:
		m.Answer = append(m.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
			AAAA: net.IPv6zero,
		})
	default: // A et les autres
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.IPv4zero,
		})
	}
	_ = w.WriteMsg(m)
}

// respondFail répond un échec (SERVFAIL) en cas d'erreur upstream.
func (s *Shield) respondFail(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Rcode = dns.RcodeServerFailure
	_ = w.WriteMsg(m)
}

// canonicalDomain normalise un nom DNS (minuscules, sans point final).
func canonicalDomain(name string) string {
	d := name
	if len(d) > 0 && d[len(d)-1] == '.' {
		d = d[:len(d)-1]
	}
	// lowercase simple
	out := make([]byte, len(d))
	for i := 0; i < len(d); i++ {
		c := d[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
