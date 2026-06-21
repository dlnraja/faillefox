package updater

import (
	"context"
	"testing"

	"github.com/dlnraja/faillefox/internal/core"
)

// TestParseHostsBasic vérifie le parsing d'un contenu hosts simple.
func TestParseHostsBasic(t *testing.T) {
	u := New(core.NewBlocklist())
	content := `# commentaire
0.0.0.0 ads.example.com
127.0.0.1 tracker.evil.org
localhost
192.168.1.1 myrouter
clean.site.com
`
	n := u.parseHosts(content)
	// 4 entrées valides attendues : ads.example.com, tracker.evil.org,
	// myrouter, clean.site.com (pas localhost).
	if n != 4 {
		t.Errorf("entrées attendues: 4, got %d", n)
	}
	if !u.blocklist.Contains("ads.example.com") {
		t.Error("ads.example.com devrait être bloqué")
	}
	if u.blocklist.Contains("localhost") {
		t.Error("localhost ne doit jamais être bloqué")
	}
}

// TestShortURL vérifie le raccourcissement d'URL pour les logs.
func TestShortURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts", "raw.githubusercontent.com"},
		{"https://oisd.nl/downloads/wildcardLight.txt", "oisd.nl"},
		{"simple", "simple"},
	}
	for _, tc := range tests {
		if got := shortURL(tc.in); got != tc.want {
			t.Errorf("shortURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFetchOnceOffline vérifie que FetchOnce gère proprement les échecs
// réseau (CI sans réseau) sans planter.
func TestFetchOnceOffline(t *testing.T) {
	u := New(core.NewBlocklist())
	// On remplace les sources par une URL inexistante pour forcer l'échec.
	u.dnsSources = []string{"http://127.0.0.1:1/nonexistent"}
	n, err := u.FetchOnce(context.Background())
	// FetchOnce ne renvoie pas d'erreur (elle logge et continue), mais le
	// compte doit être 0.
	if err != nil {
		t.Errorf("FetchOnce ne devrait pas propager l'erreur: %v", err)
	}
	if n != 0 {
		t.Errorf("compte attendu 0 en cas d'échec réseau, got %d", n)
	}
}
