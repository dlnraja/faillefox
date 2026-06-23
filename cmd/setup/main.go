// Command setup vérifie la configuration de Faillefox en une seule commande.
//
// Valide : token d'auth, clés API antivirus, SignPath, ClamAV, connectivité.
//
// Lancement : setup
package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/dlnraja/faillefox/internal/setup"
)

func main() {
	var dataDir = flag.String("data", defaultDataDir(), "répertoire de données")
	flag.Parse()

	results := setup.RunAll(*dataDir)
	setup.Print(results)

	// Exit code : 0 si tout OK, 1 si échecs critiques.
	for _, r := range results {
		if r.Status == "fail" {
			os.Exit(1)
		}
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".faillefox")
}
