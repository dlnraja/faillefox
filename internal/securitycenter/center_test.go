package securitycenter

import "testing"

// TestNewHasAllProtections vérifie que New crée toutes les protections
// attendues, en StatusInactive par défaut.
func TestNewHasAllProtections(t *testing.T) {
	c := New()
	states := c.States()
	if len(states) == 0 {
		t.Fatal("le centre devrait déclarer des protections")
	}
	// Toutes devraient être inactive au départ.
	for _, s := range states {
		if s.Status != StatusInactive {
			t.Errorf("protection %s devrait être inactive au départ, got %s", s.ID, s.Status)
		}
	}
}

// TestSetStatus vérifie la mise à jour d'un statut.
func TestSetStatus(t *testing.T) {
	c := New()
	c.SetStatus(ProtFirewall, StatusActive)
	states := c.States()
	for _, s := range states {
		if s.ID == ProtFirewall && s.Status != StatusActive {
			t.Errorf("firewall devrait être active, got %s", s.Status)
		}
	}
}

// TestSummary vérifie le calcul du résumé et du score.
func TestSummary(t *testing.T) {
	c := New()
	total := len(c.States())
	c.SetStatus(ProtFirewall, StatusActive)
	c.SetStatus(ProtDNS, StatusActive)
	c.SetStatus(ProtAntiAds, StatusLimited)

	s := c.GetSummary()
	if s.Total != total {
		t.Errorf("total attendu %d, got %d", total, s.Total)
	}
	if s.Active != 2 {
		t.Errorf("actives attendues 2, got %d", s.Active)
	}
	if s.Limited != 1 {
		t.Errorf("limitées attendues 1, got %d", s.Limited)
	}
	// Score : 2*100 + 1*50 = 250, divisé par total.
	expected := (2*100 + 1*50) / total
	if s.Score != expected {
		t.Errorf("score attendu %d, got %d", expected, s.Score)
	}
}

// TestSummaryAllActive vérifie le score maximal (100%).
func TestSummaryAllActive(t *testing.T) {
	c := New()
	for _, p := range allProtections() {
		c.SetStatus(p.ID, StatusActive)
	}
	s := c.GetSummary()
	if s.Score != 100 {
		t.Errorf("score avec tout actif attendu 100, got %d", s.Score)
	}
	if s.Inactive != 0 {
		t.Errorf("aucune inactive attendue, got %d", s.Inactive)
	}
}

// TestSetStats vérifie l'attachement de statistiques.
func TestSetStats(t *testing.T) {
	c := New()
	c.SetStats(ProtDNS, map[string]int{"bloques": 1500})
	for _, s := range c.States() {
		if s.ID == ProtDNS {
			if s.Stats["bloques"] != 1500 {
				t.Errorf("stats bloquées attendues 1500, got %d", s.Stats["bloques"])
			}
		}
	}
}

// TestMarkEvent vérifie l'horodatage du dernier événement.
func TestMarkEvent(t *testing.T) {
	c := New()
	c.MarkEvent(ProtFirewall)
	for _, s := range c.States() {
		if s.ID == ProtFirewall {
			if s.LastEvent == nil {
				t.Error("LastEvent devrait être défini après MarkEvent")
			}
		}
	}
}

// TestAllProtectionsContainKey vérifie que les catégories attendues existent.
func TestAllProtectionsContainKey(t *testing.T) {
	prots := allProtections()
	ids := make(map[Protection]bool)
	for _, p := range prots {
		ids[p.ID] = true
	}
	required := []Protection{
		ProtFirewall, ProtDNS, ProtAntiAds, ProtAntiTrackers,
		ProtAntiMalware, ProtAntiAdware, ProtAntiPhishing,
		ProtAVScanner, ProtYARAScanner, ProtCVEFeed, ProtThreatIntel,
	}
	for _, r := range required {
		if !ids[r] {
			t.Errorf("protection requise manquante: %s", r)
		}
	}
}
