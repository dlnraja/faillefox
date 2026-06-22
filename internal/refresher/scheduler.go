// Package refresher orchestre le rafraîchissement périodique de TOUS les
// référentiels externes de Faillefox, de façon unifiée et autonome.
//
// Référentiels gérés :
//   - Listes DNS anti-pubs/trackers/malwares (via updater.Updater)
//   - Base CVE NVD (via cvefeed.Feed)
//   - IOC threat intel Abuse.ch + OTX (via threatintel.Aggregator)
//   - Règles YARA publiques (via yarascan.Scanner — rechargement)
//
// Chaque référentiel a sa propre cadence optimale :
//   - DNS       : 6h  (malwares émergents)
//   - CVE       : 6h  (NVD publie en continu)
//   - Threat    : 3h  (IOC APT bougent vite)
//   - YARA      : 24h (règles publiques, moins volatiles)
//
// Le scheduler tourne en arrière-plan dès le démarrage. Il expose son état
// via Status() pour l'UI (quand a-t-on rafraîchi quoi, prochaine échéance).
package refresher

import (
	"context"
	"log"
	"sync"
	"time"
)

// Source identifie un référentiel rafraîchissable.
type Source string

const (
	SrcDNS    Source = "dns"
	SrcCVE    Source = "cve"
	SrcThreat Source = "threat_intel"
	SrcYARA   Source = "yara"
)

// RefreshFn est la fonction de rafraîchissement d'un référentiel.
// Elle renvoie le nombre d'éléments ajoutés/mis à jour, ou une erreur.
type RefreshFn func(ctx context.Context) (int, error)

// sourceState est l'état d'un référentiel (pour l'observabilité).
type sourceState struct {
	name         Source
	last         time.Time
	next         time.Time
	items        int
	lastError    string
	cycle        int
}

// Scheduler orchestre le rafraîchissement périodique de toutes les sources.
type Scheduler struct {
	mu      sync.RWMutex
	sources map[Source]*sourceState
	fns     map[Source]RefreshFn
	every   map[Source]time.Duration
}

// New crée un scheduler vide.
func New() *Scheduler {
	return &Scheduler{
		sources: make(map[Source]*sourceState),
		fns:     make(map[Source]RefreshFn),
		every:   make(map[Source]time.Duration),
	}
}

// Register ajoute un référentiel au scheduler avec sa fonction de refresh
// et sa cadence.
func (s *Scheduler) Register(src Source, fn RefreshFn, every time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources[src] = &sourceState{name: src}
	s.fns[src] = fn
	s.every[src] = every
}

// Start lance la boucle de rafraîchissement. Bloquant ; à lancer dans une
// goroutine. Au démarrage, chaque source est rafraîchie une fois (en parallèle),
// puis à sa cadence propre.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.RLock()
	sources := make([]Source, 0, len(s.sources))
	for src := range s.sources {
		sources = append(sources, src)
	}
	s.mu.RUnlock()

	// Une goroutine par source : chacune gère sa propre cadence.
	var wg sync.WaitGroup
	for _, src := range sources {
		wg.Add(1)
		go func(src Source) {
			defer wg.Done()
			s.runSourceLoop(ctx, src)
		}(src)
	}
	wg.Wait()
}

// runSourceLoop rafraîchit une source à intervalle régulier.
func (s *Scheduler) runSourceLoop(ctx context.Context, src Source) {
	// Refresh initial au démarrage.
	s.refreshOne(ctx, src)

	every := s.every[src]
	if every <= 0 {
		return
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshOne(ctx, src)
		}
	}
}

// refreshOne exécute une fonction de refresh et met à jour l'état.
func (s *Scheduler) refreshOne(ctx context.Context, src Source) {
	s.mu.RLock()
	fn := s.fns[src]
	every := s.every[src]
	state := s.sources[src]
	s.mu.RUnlock()
	if fn == nil || state == nil {
		return
	}

	log.Printf("[refresher] rafraîchissement %s...", src)
	n, err := fn(ctx)

	s.mu.Lock()
	state.last = time.Now()
	state.next = state.last.Add(every)
	state.items = n
	state.cycle++
	if err != nil {
		state.lastError = err.Error()
		log.Printf("[refresher] %s: %v", src, err)
	} else {
		state.lastError = ""
		log.Printf("[refresher] %s: %d éléments", src, n)
	}
	s.mu.Unlock()
}

// SourceStatus est l'état public d'une source (pour l'API/UI).
type SourceStatus struct {
	Name      Source    `json:"name"`
	Last      time.Time `json:"last_refresh"`
	Next      time.Time `json:"next_refresh"`
	Items     int       `json:"items"`
	LastError string    `json:"last_error,omitempty"`
	Cycle     int       `json:"cycle"`
	Every     string    `json:"every"`
}

// Status renvoie l'état de toutes les sources (pour l'UI).
func (s *Scheduler) Status() []SourceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SourceStatus, 0, len(s.sources))
	for src, st := range s.sources {
		out = append(out, SourceStatus{
			Name:      st.name,
			Last:      st.last,
			Next:      st.next,
			Items:     st.items,
			LastError: st.lastError,
			Cycle:     st.cycle,
			Every:     s.every[src].String(),
		})
	}
	return out
}
