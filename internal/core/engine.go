package core

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Engine est le moteur de décision du pare-feu. Il détient les règles,
// la politique par défaut, le journal et les abonnés aux événements.
//
// Il est CONCURRENT-SAFE : plusieurs backends et l'API HTTP peuvent
// l'utiliser simultanément.
type Engine struct {
	mu sync.RWMutex

	store     Store
	rules     []Rule
	defaults  Decision // action quand aucune règle ne matche

	// journal borné en mémoire (ring) pour l'affichage temps réel
	journal   []Event
	journalMu sync.Mutex
	journalCap int

	// abonnés au flux d'événements (UI via SSE)
	subsMu sync.Mutex
	subs   map[string]chan Event
}

// NewEngine crée un moteur avec un store de persistance donné.
// La politique par défaut est "ask" (demander à l'utilisateur) tant que
// l'utilisateur n'a pas configuré de politique plus stricte.
func NewEngine(store Store) *Engine {
	return &Engine{
		store:      store,
		defaults:   DecisionAsk,
		journalCap: 500,
		subs:       make(map[string]chan Event),
	}
}

// Load lit les règles persistées. À appeler au démarrage.
func (e *Engine) Load() error {
	rules, defaults, err := e.store.Load()
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = rules
	if defaults != "" {
		e.defaults = defaults
	}
	e.mu.Unlock()
	return nil
}

// Save persiste l'état actuel.
func (e *Engine) Save() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.store.Save(e.rules, e.defaults)
}

// Rules renvoie une copie des règles courantes.
func (e *Engine) Rules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Rule, len(e.rules))
	copy(out, e.rules)
	return out
}

// DefaultDecision renvoie la politique par défaut.
func (e *Engine) DefaultDecision() Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defaults
}

// SetDefaultDecision met à jour la politique par défaut et persiste.
func (e *Engine) SetDefaultDecision(d Decision) error {
	e.mu.Lock()
	e.defaults = d
	e.mu.Unlock()
	return e.Save()
}

// AddRule ajoute une règle et persiste. L'ID et le timestamp sont remplis
// si absents, et la règle complète (avec son ID) est renvoyée.
func (e *Engine) AddRule(r Rule) (Rule, error) {
	if r.ID == "" {
		r.ID = newID()
	}
	r.Created = time.Now()
	e.mu.Lock()
	e.rules = append(e.rules, r)
	e.mu.Unlock()
	return r, e.Save()
}

// DeleteRule supprime une règle par ID et persiste.
func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			break
		}
	}
	e.mu.Unlock()
	return e.Save()
}

// Decide calcule la décision pour une connexion, journalise et notifie.
// C'est la fonction appelée par les backends pour chaque connexion
// sortante interceptée.
func (e *Engine) Decide(c Connection) Decision {
	e.mu.RLock()
	matched := Decision("")
	reason := "default"
	for _, r := range e.rules {
		if r.Match(&c) {
			matched = r.Action
			reason = "rule:" + r.ID
			break
		}
	}
	decision := matched
	if decision == "" {
		decision = e.defaults
	}
	e.mu.RUnlock()

	ev := Event{
		ID:         newID(),
		Connection: c,
		Decision:   decision,
		Reason:     reason,
		At:         time.Now(),
	}
	e.appendEvent(ev)
	return decision
}

// journal -------------------------------------------------------------------

func (e *Engine) appendEvent(ev Event) {
	e.journalMu.Lock()
	e.journal = append(e.journal, ev)
	if len(e.journal) > e.journalCap {
		e.journal = e.journal[len(e.journal)-e.journalCap:]
	}
	e.journalMu.Unlock()

	e.subsMu.Lock()
	for _, ch := range e.subs {
		// Non bloquant : si l'abonné ne consomme pas, on abandonne l'événement.
		select {
		case ch <- ev:
		default:
		}
	}
	e.subsMu.Unlock()
}

// RecentEvents renvoie les derniers événements du journal (les plus récents
// en dernier), limité à n.
func (e *Engine) RecentEvents(n int) []Event {
	e.journalMu.Lock()
	defer e.journalMu.Unlock()
	if n > len(e.journal) || n <= 0 {
		n = len(e.journal)
	}
	out := make([]Event, n)
	copy(out, e.journal[len(e.journal)-n:])
	return out
}

// Subscribe enregistre un canal récepteur d'événements en direct.
// Renvoie un ID et le canal. Désabonnement via Unsubscribe(id).
func (e *Engine) Subscribe(buf int) (string, <-chan Event) {
	id := newID()
	ch := make(chan Event, buf)
	e.subsMu.Lock()
	e.subs[id] = ch
	e.subsMu.Unlock()
	return id, ch
}

// Unsubscribe supprime un abonné.
func (e *Engine) Unsubscribe(id string) {
	e.subsMu.Lock()
	if ch, ok := e.subs[id]; ok {
		delete(e.subs, id)
		close(ch)
	}
	e.subsMu.Unlock()
}

// newID génère un identifiant aléatoire court (16 hex chars).
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
