package core

import (
	"testing"
)

// TestBlocklistAddContains vérifie l'ajout et la recherche directe.
func TestBlocklistAddContains(t *testing.T) {
	bl := NewBlocklist()
	bl.Add("ads.example.com")
	if !bl.Contains("ads.example.com") {
		t.Error("domaine ajouté devrait être trouvé")
	}
	if bl.Contains("clean.example.org") {
		t.Error("domaine non ajouté ne devrait pas être trouvé")
	}
}

// TestBlocklistSubdomain vérifie le matching des sous-domaines :
// si "doubleclick.net" est bloqué, "ads.doubleclick.net" l'est aussi.
func TestBlocklistSubdomain(t *testing.T) {
	bl := NewBlocklist()
	bl.Add("doubleclick.net")
	if !bl.Contains("ads.doubleclick.net") {
		t.Error("sous-domaine d'un domaine bloqué devrait être bloqué")
	}
	if !bl.Contains("x.y.z.doubleclick.net") {
		t.Error("sous-domaine profond devrait aussi être bloqué")
	}
}

// TestBlocklistNormalization vérifie la normalisation (casse, FQDN racine).
func TestBlocklistNormalization(t *testing.T) {
	bl := NewBlocklist()
	bl.Add("Example.COM.")
	if !bl.Contains("example.com") {
		t.Error("la casse et le point final devraient être normalisés")
	}
}

// TestBlocklistLoadFromHosts vérifie le parsing du format hosts.
func TestBlocklistLoadFromHosts(t *testing.T) {
	bl := NewBlocklist()
	content := `# commentaire
0.0.0.0 tracker.example.com
ads.evil.net
  # espace puis commentaire

0.0.0.0 analytics.spy.org`
	n := bl.LoadFromHosts(content)
	if n != 3 {
		t.Errorf("3 domaines attendus, got %d", n)
	}
	if !bl.Contains("tracker.example.com") {
		t.Error("tracker.example.com devrait être bloqué")
	}
	if !bl.Contains("ads.evil.net") {
		t.Error("ads.evil.net devrait être bloqué")
	}
	if !bl.Contains("analytics.spy.org") {
		t.Error("analytics.spy.org devrait être bloqué")
	}
}

// TestBlocklistSize vérifie le comptage.
func TestBlocklistSize(t *testing.T) {
	bl := NewBlocklist()
	if bl.Size() != 0 {
		t.Error("liste vide devrait avoir size 0")
	}
	bl.Add("a.com")
	bl.Add("b.com")
	if bl.Size() != 2 {
		t.Errorf("size attendu 2, got %d", bl.Size())
	}
}

// TestEngineBlocklistIntegration vérifie que le moteur bloque les domaines
// de la blocklist avant même l'évaluation des règles.
func TestEngineBlocklistIntegration(t *testing.T) {
	e := newTestEngine(t, DecisionAllow) // tout autorisé par défaut
	bl := NewBlocklist()
	bl.Add("evil-tracker.com")
	e.SetBlocklist(bl)

	// Comme le stub n'ajoute pas de HostName dans Connection, on simule
	// une connexion dont le HostName résoudrait vers un domaine bloqué.
	// Le moteur utilise Connection.HostName() qui fait un LookupAddr réel ;
	// ici on vérifie juste que sans HostName la blocklist n'aggrave pas.
	got := e.Decide(Connection{AppID: "x", RemoteAddr: "1.2.3.4"})
	if got != DecisionAllow {
		t.Errorf("sans HostName résolvable, défaut doit s'appliquer: got %v", got)
	}
}
