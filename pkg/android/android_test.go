package android

import "testing"

// TestEngineNew vérifie la création d'un moteur vide.
func TestEngineNew(t *testing.T) {
	e := NewEngine()
	if e.RuleCount() != 0 {
		t.Errorf("nouveau moteur devrait avoir 0 règle, got %d", e.RuleCount())
	}
}

// TestEngineDecideDefault vérifie le comportement par défaut (allow).
func TestEngineDecideDefault(t *testing.T) {
	e := NewEngine()
	d := e.Decide("unknown.app", "tcp", "1.2.3.4", 443)
	if d != "allow" {
		t.Errorf("app sans règle devrait être allow, got %s", d)
	}
}

// TestEngineSetRuleDeny vérifie qu'une règle deny est appliquée.
func TestEngineSetRuleDeny(t *testing.T) {
	e := NewEngine()
	e.SetRule("evil.app", "deny")
	if d := e.Decide("evil.app", "tcp", "1.2.3.4", 443); d != "deny" {
		t.Errorf("evil.app devrait être deny, got %s", d)
	}
}

// TestEngineSetRuleAllow vérifie qu'une règle allow est appliquée.
func TestEngineSetRuleAllow(t *testing.T) {
	e := NewEngine()
	e.SetRule("good.app", "deny")
	e.SetRule("good.app", "allow")
	if d := e.Decide("good.app", "tcp", "1.2.3.4", 443); d != "allow" {
		t.Errorf("good.app devrait être allow après override, got %s", d)
	}
}

// TestEngineSetRuleInvalid vérifie qu'une décision invalide est ignorée.
func TestEngineSetRuleInvalid(t *testing.T) {
	e := NewEngine()
	e.SetRule("test.app", "BLOCK!") // invalide, doit être ignoré
	if d := e.Decide("test.app", "tcp", "1.2.3.4", 443); d != "allow" {
		t.Errorf("règle invalide ignorée, doit rester allow, got %s", d)
	}
}

// TestEngineRemoveRule vérifie la suppression d'une règle.
func TestEngineRemoveRule(t *testing.T) {
	e := NewEngine()
	e.SetRule("temp.app", "deny")
	e.RemoveRule("temp.app")
	if d := e.Decide("temp.app", "tcp", "1.2.3.4", 443); d != "allow" {
		t.Errorf("après remove, doit revenir à allow, got %s", d)
	}
}

// TestEngineRuleCount vérifie le comptage.
func TestEngineRuleCount(t *testing.T) {
	e := NewEngine()
	e.SetRule("a", "deny")
	e.SetRule("b", "allow")
	e.SetRule("c", "deny")
	if e.RuleCount() != 3 {
		t.Errorf("attendu 3 règles, got %d", e.RuleCount())
	}
}

// TestVersion vérifie que Version renvoie une chaîne non vide.
func TestVersion(t *testing.T) {
	if Version() == "" {
		t.Error("Version() ne devrait pas être vide")
	}
}

// TestIsPrivateIP vérifie la détection des IP privées.
func TestIsPrivateIP(t *testing.T) {
	privates := []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1"}
	for _, ip := range privates {
		if !IsPrivateIP(ip) {
			t.Errorf("%s devrait être privée", ip)
		}
	}
	publics := []string{"8.8.8.8", "1.1.1.1", "203.0.113.1"}
	for _, ip := range publics {
		if IsPrivateIP(ip) {
			t.Errorf("%s ne devrait pas être privée", ip)
		}
	}
}

// TestIsPrivateIPInvalid vérifie les entrées invalides.
func TestIsPrivateIPInvalid(t *testing.T) {
	if IsPrivateIP("not-an-ip") {
		t.Error("entrée invalide ne devrait pas être privée")
	}
	if IsPrivateIP("") {
		t.Error("chaîne vide ne devrait pas être privée")
	}
}

// TestLookupHostInvalid vérifie qu'une IP invalide renvoie "".
func TestLookupHostInvalid(t *testing.T) {
	if LookupHost("not-an-ip") != "" {
		t.Error("IP invalide devrait renvoyer vide")
	}
}
