// Bindings gomobile exposant le cœur Faillefox à l'app Android Kotlin.
// gomobile n'accepte que des types simples en paramètres/retours ; on
// encapsule donc le moteur derrière une API string-based.
package android

import (
	"encoding/json"

	"github.com/dlnraja/faillefox/internal/core"
)

// EngineWrapper est l'objet exposé à Kotlin via gomobile.
type EngineWrapper struct {
	engine *core.Engine
}

// NewEngine crée un wrapper avec un store en mémoire (sur Android, la
// persistance est gérée côté Kotlin via SharedPreferences).
func NewEngine() *EngineWrapper {
	store := &memoryStore{}
	return &EngineWrapper{engine: core.NewEngine(store)}
}

// DecideJSON reçoit une connexion sérialisée en JSON et renvoie la décision
// ("allow" / "deny" / "ask"). Côté Kotlin, on appelle via gomobile :
//
//   val wrapper = Faillefox.NewEngine()
//   val decision = wrapper.decideJSON(jsonConn) // "deny" par ex.
func (w *EngineWrapper) DecideJSON(connJSON string) string {
	var c core.Connection
	if err := json.Unmarshal([]byte(connJSON), &c); err != nil {
		return string(core.DecisionAllow) // safe default
	}
	return string(w.engine.Decide(c))
}

// AddRuleJSON ajoute une règle depuis sa forme JSON.
func (w *EngineWrapper) AddRuleJSON(ruleJSON string) string {
	var r core.Rule
	if err := json.Unmarshal([]byte(ruleJSON), &r); err != nil {
		return ""
	}
	saved, _ := w.engine.AddRule(r)
	b, _ := json.Marshal(saved)
	return string(b)
}

// RulesJSON renvoie toutes les règles en JSON.
func (w *EngineWrapper) RulesJSON() string {
	b, _ := json.Marshal(w.engine.Rules())
	return string(b)
}

// SetDefault change la politique par défaut ("allow"/"deny"/"ask").
func (w *EngineWrapper) SetDefault(decision string) {
	_ = w.engine.SetDefaultDecision(core.Decision(decision))
}

// memoryStore : store en mémoire pour Android (la persistance est gérée
// côté Kotlin). Implémente core.Store.
type memoryStore struct {
	rules    []core.Rule
	defaults core.Decision
}

func (m *memoryStore) Load() ([]core.Rule, core.Decision, error) {
	return m.rules, m.defaults, nil
}
func (m *memoryStore) Save(rules []core.Rule, defaults core.Decision) error {
	m.rules = rules
	m.defaults = defaults
	return nil
}
