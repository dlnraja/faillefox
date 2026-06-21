package correlate

import (
	"testing"

	"github.com/dlnraja/faillefox/internal/core"
	"github.com/dlnraja/faillefox/internal/threatintel"
)

// TestEvaluateConnectionNoSignal vérifie qu'une connexion sans signal renvoie nil.
func TestEvaluateConnectionNoSignal(t *testing.T) {
	agg := threatintel.New()
	c := New(agg)
	conn := core.Connection{RemoteAddr: "8.8.8.8", AppName: "chrome"}
	if alert := c.EvaluateConnection(conn); alert != nil {
		t.Errorf("connexion propre ne doit pas générer d'alerte, got %+v", alert)
	}
}

// TestEvaluateConnectionThreatIntel vérifie qu'une IP connue produit une alerte.
func TestEvaluateConnectionThreatIntel(t *testing.T) {
	agg := threatintel.New()
	agg.Add(threatintel.IOC{Value: "1.2.3.4", Source: "abuse.ch"})
	agg.Add(threatintel.IOC{Value: "1.2.3.4", Source: "otx"})

	c := New(agg)
	conn := core.Connection{RemoteAddr: "1.2.3.4", AppName: "chrome"}
	alert := c.EvaluateConnection(conn)
	if alert == nil {
		t.Fatal("IP connue dans 2 sources devrait produire une alerte")
	}
	// 2 sources * 30 = 60 points -> severityMedium.
	if alert.Score != 60 {
		t.Errorf("score attendu 60 (2 sources * 30), got %d", alert.Score)
	}
	if alert.Severity != SeverityMedium {
		t.Errorf("severity attendue medium, got %s", alert.Severity)
	}
}

// TestEvaluateConnectionCVEBoost vérifie le boost si le logiciel a une CVE.
func TestEvaluateConnectionCVEBoost(t *testing.T) {
	agg := threatintel.New()
	agg.Add(threatintel.IOC{Value: "9.9.9.9", Source: "abuse.ch"})

	c := New(agg)
	c.SetCVEIndex(map[string]bool{"curl": true})

	conn := core.Connection{RemoteAddr: "9.9.9.9", AppName: "/usr/bin/curl"}
	alert := c.EvaluateConnection(conn)
	if alert == nil {
		t.Fatal("alerte attendue (IOC + CVE)")
	}
	// 1 source * 30 + 40 CVE = 70.
	if alert.Score != 70 {
		t.Errorf("score attendu 70 (30 IOC + 40 CVE), got %d", alert.Score)
	}
}

// TestEvaluateConnectionCritical vérifie le score critique (IOC x3 + CVE + public).
func TestEvaluateConnectionCritical(t *testing.T) {
	agg := threatintel.New()
	for _, s := range []string{"abuse.ch", "otx", "misp"} {
		agg.Add(threatintel.IOC{Value: "10.0.0.1", Source: s})
	}

	c := New(agg)
	c.SetCVEIndex(map[string]bool{"chrome": true})
	c.SetProfile(core.ProfilePublic)

	conn := core.Connection{RemoteAddr: "10.0.0.1", AppName: "chrome"}
	alert := c.EvaluateConnection(conn)
	if alert == nil {
		t.Fatal("alerte critique attendue")
	}
	// 3 sources * 30 + 40 CVE + 20 public = 150.
	if alert.Score != 150 {
		t.Errorf("score attendu 150 (90+40+20), got %d", alert.Score)
	}
	if alert.Severity != SeverityCritical {
		t.Errorf("severity attendue critical, got %s", alert.Severity)
	}
}

// TestLowerAppName vérifie la normalisation des noms d'app.
func TestLowerAppName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Google Chrome", "google chrome"},
		{"/usr/bin/curl", "curl"},
		{"C:\\Windows\\chrome.exe", "chrome"},
		{"bash", "bash"},
	}
	for _, tc := range tests {
		if got := lowerAppName(tc.in); got != tc.want {
			t.Errorf("lowerAppName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSeverityFromScore vérifie le mapping score -> severity.
func TestSeverityFromScore(t *testing.T) {
	tests := []struct {
		score int
		want  Severity
	}{
		{29, SeverityLow}, // en pratique < 30 ne génère pas d'alerte, mais test du mapping
		{50, SeverityLow},
		{70, SeverityMedium},
		{100, SeverityHigh},
		{150, SeverityCritical},
	}
	for _, tc := range tests {
		if got := severityFromScore(tc.score); got != tc.want {
			t.Errorf("severityFromScore(%d) = %s, want %s", tc.score, got, tc.want)
		}
	}
}
