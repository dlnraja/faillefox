// Command faillefox-gui est l'interface desktop native de Faillefox.
//
// C'est un binaire SÉPARÉ de faillefox (qui reste pur Go sans CGO) car
// Fyne nécessite CGO + OpenGL. Le GUI parle à l'API loopback du démon
// faillefox tournant en arrière-plan.
//
// Lancement :
//
//	# 1. Démarrer le démon (en service ou en console)
//	faillefox -dns -cve -threat-intel
//
//	# 2. Lancer le GUI dans un autre terminal
//	faillefox-gui
//	# ou avec un port/token spécifique :
//	faillefox-gui -port 8443 -token <votre-token>
//
// Le GUI récupère automatiquement le token depuis ~/.faillefox/token si
// non fourni en argument.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dlnraja/faillefox/internal/auth"
	"github.com/dlnraja/faillefox/internal/desktop"
)

func main() {
	var (
		port      = flag.Int("port", 8443, "port du démon Faillefox (loopback)")
		tokenArg  = flag.String("token", "", "token d'authentification API (défaut: ~/.faillefox/token)")
		dataDir   = flag.String("data", defaultDataDir(), "répertoire de données")
	)
	flag.Parse()

	// Récupération du token (argument ou fichier).
	token := *tokenArg
	if token == "" {
		t, err := auth.LoadOrCreate(filepath.Join(*dataDir, "token"))
		if err != nil {
			log.Printf("[warn] token: %v (GUI sans auth — fonctionnalités limitées)", err)
		} else {
			token = t.Value()
		}
	}

	fmt.Println("🦊 Faillefox GUI — connexion au démon sur 127.0.0.1:", *port)
	if err := desktop.Run(*port, token); err != nil {
		log.Fatalf("[fatal] GUI: %v", err)
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".faillefox")
}
