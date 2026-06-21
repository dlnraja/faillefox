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
// réseau (CI sans réseau) sans planter. Une erreur réseau est renvoyée
// mais l'état interne reste cohérent.
func TestFetchOnceOffline(t *testing.T) {
	u := New(core.NewBlocklist())
	// On remplace les sources par une URL inexistante pour forcer l'échec.
	u.dnsSources = []string{"http://127.0.0.1:1/nonexistent"}
	n, err := u.FetchOnce(context.Background())
	// Une erreur réseau est légitimement renvoyée.
	if err == nil {
		t.Error("FetchOnce devrait renvoyer une erreur en cas d'échec réseau")
	}
	// Aucun domaine n'a pu être téléchargé.
	if n != 0 {
		t.Errorf("compte attendu 0 en cas d'échec réseau, got %d", n)
	}
	// Mais l'état interne doit quand même être mis à jour (cycle incrémenté).
	if u.Status().CycleCount != 1 {
		t.Error("le cycle devrait être incrémenté même en échec")
	}
}

// TestStatusAfterFetch vérifie que Status reflète l'état après un FetchOnce.
func TestStatusAfterFetch(t *testing.T) {
	u := New(core.NewBlocklist())
	u.dnsSources = []string{"http://127.0.0.1:1/nonexistent"} // échec voulu
	_, _ = u.FetchOnce(context.Background())

	st := u.Status()
	// Après un fetch échoué, lastError doit être renseigné, lastFetch non nulle.
	if st.LastFetch.IsZero() {
		t.Error("lastFetch devrait être non nulle après FetchOnce")
	}
	if st.LastError == "" {
		t.Error("lastError devrait être renseignée après échec réseau")
	}
	if st.CycleCount != 1 {
		t.Errorf("cycle attendu 1, got %d", st.CycleCount)
	}
	if st.UpdateEvery == "" {
		t.Error("updateEvery devrait être renseignée")
	}
}

// TestStatusInitiallyClean vérifie l'état initial (avant tout fetch).
func TestStatusInitiallyClean(t *testing.T) {
	u := New(core.NewBlocklist())
	st := u.Status()
	if !st.LastFetch.IsZero() {
		t.Error("lastFetch devrait être zéro initialement")
	}
	if st.LastError != "" {
		t.Error("lastError devrait être vide initialement")
	}
	if st.CycleCount != 0 {
		t.Errorf("cycle attendu 0 initialement, got %d", st.CycleCount)
	}
}
