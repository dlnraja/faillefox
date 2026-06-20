package core

import (
	"testing"
)

// TestRuleMatch vérifie la logique de correspondance d'une règle avec une
// connexion. C'est le cœur du filtrage : une règle ne doit matcher QUE si
// tous ses champs non vides correspondent.
func TestRuleMatch(t *testing.T) {
	conn := Connection{
		AppID:      "app.exe",
		Protocol:   ProtocolTCP,
		RemoteAddr: "1.2.3.4",
		RemotePort: 443,
	}

	tests := []struct {
		name string
		rule Rule
		want bool
	}{
		{"règle vide (catch-all)", Rule{}, true},
		{"app seule qui matche", Rule{AppID: "app.exe"}, true},
		{"app qui ne matche pas", Rule{AppID: "autre.exe"}, false},
		{"proto qui matche", Rule{Protocol: ProtocolTCP}, true},
		{"proto qui ne matche pas", Rule{Protocol: ProtocolUDP}, false},
		{"port qui matche", Rule{Port: 443}, true},
		{"port qui ne matche pas", Rule{Port: 80}, false},
		{"ip qui matche", Rule{IP: "1.2.3.4"}, true},
		{"ip qui ne matche pas", Rule{IP: "9.9.9.9"}, false},
		{"combinaison complète qui matche",
			Rule{AppID: "app.exe", Protocol: ProtocolTCP, Port: 443, IP: "1.2.3.4"}, true},
		{"combinaison partielle (app faux)",
			Rule{AppID: "x", Protocol: ProtocolTCP, Port: 443}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.Match(&conn); got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDecideDefault vérifie que la politique par défaut s'applique quand
// aucune règle ne matche.
func TestDecideDefault(t *testing.T) {
	e := newTestEngine(t, DecisionDeny)

	got := e.Decide(Connection{AppID: "x"})
	if got != DecisionDeny {
		t.Errorf("décision par défaut = %v, want deny", got)
	}
}

// TestDecideFirstMatchingRuleWins vérifie que la première règle qui matche
// gagne, et que les suivantes sont ignorées (ordre = priorité).
func TestDecideFirstMatchingRuleWins(t *testing.T) {
	e := newTestEngine(t, DecisionAllow)
	// Une règle deny générique, puis une allow plus spécifique.
	// Comme deny vient en premier et matche, deny doit l'emporter.
	if _, err := e.AddRule(Rule{AppID: "a", Action: DecisionDeny}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddRule(Rule{AppID: "a", Port: 443, Action: DecisionAllow}); err != nil {
		t.Fatal(err)
	}
	got := e.Decide(Connection{AppID: "a", RemotePort: 443})
	if got != DecisionDeny {
		t.Errorf("première règle devrait gagner: got %v, want deny", got)
	}
}

// TestDecideSpecificRuleOverridesDefault vérifie qu'une règle précise
// prend le pas sur la politique par défaut.
func TestDecideSpecificRuleOverridesDefault(t *testing.T) {
	e := newTestEngine(t, DecisionAllow) // tout autorisé par défaut
	if _, err := e.AddRule(Rule{AppID: "evil.exe", Action: DecisionDeny}); err != nil {
		t.Fatal(err)
	}
	if got := e.Decide(Connection{AppID: "evil.exe"}); got != DecisionDeny {
		t.Errorf("règle spécifique devrait bloquer: got %v, want deny", got)
	}
	if got := e.Decide(Connection{AppID: "good.exe"}); got != DecisionAllow {
		t.Errorf("app sans règle = défaut: got %v, want allow", got)
	}
}

// TestDecideJournalizesEvent vérifie que chaque décision produit un
// événement dans le journal.
func TestDecideJournalizesEvent(t *testing.T) {
	e := newTestEngine(t, DecisionAllow)
	e.Decide(Connection{AppID: "a"})
	e.Decide(Connection{AppID: "b"})
	events := e.RecentEvents(0)
	if len(events) != 2 {
		t.Errorf("journal devrait contenir 2 événements, en contient %d", len(events))
	}
}

// TestSubscribeReceivesEvents vérifie le flux temps réel d'événements
// utilisé par le SSE de l'UI.
func TestSubscribeReceivesEvents(t *testing.T) {
	e := newTestEngine(t, DecisionAllow)
	id, ch := e.Subscribe(8)
	defer e.Unsubscribe(id)

	e.Decide(Connection{AppID: "a"})
	select {
	case ev := <-ch:
		if ev.Connection.AppID != "a" {
			t.Errorf("événement reçu pour la mauvaise app: %s", ev.Connection.AppID)
		}
	default:
		t.Error("aucun événement reçu sur le canal d'abonnement")
	}
}

// TestDeleteRule vérifie la suppression d'une règle par ID.
func TestDeleteRule(t *testing.T) {
	e := newTestEngine(t, DecisionAllow)
	r, _ := e.AddRule(Rule{AppID: "a", Action: DecisionDeny})
	if len(e.Rules()) != 1 {
		t.Fatalf("devrait y avoir 1 règle, il y en a %d", len(e.Rules()))
	}
	if err := e.DeleteRule(r.ID); err != nil {
		t.Fatal(err)
	}
	if len(e.Rules()) != 0 {
		t.Errorf("la règle n'a pas été supprimée")
	}
}

// newTestEngine crée un moteur avec un store en mémoire + une politique
// par défaut donnée. Évite la dépendance au système de fichiers.
func newTestEngine(t *testing.T, defaults Decision) *Engine {
	t.Helper()
	e := NewEngine(&memoryStore{defaults: defaults})
	if err := e.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return e
}

// memoryStore est un Store en mémoire pour les tests.
type memoryStore struct {
	rules    []Rule
	defaults Decision
}

func (m *memoryStore) Load() ([]Rule, Decision, error) { return m.rules, m.defaults, nil }
func (m *memoryStore) Save(rules []Rule, defaults Decision) error {
	m.rules = rules
	m.defaults = defaults
	return nil
}
