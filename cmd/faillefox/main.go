// Command faillefox est le point d'entrée du pare-feu Faillefox.
//
// Lancement:
//
//	faillefox                  # driver stub + UI sur http://127.0.0.1:8443
//	faillefox -driver windows-wfp
//	faillefox -port 9000
//	faillefox -list-drivers   # affiche les pilotes compilés
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dlnraja/faillefox/internal/api"
	"github.com/dlnraja/faillefox/internal/core"
	_ "github.com/dlnraja/faillefox/internal/drivers/stub" // registre le stub
)

func main() {
	var (
		driverName = flag.String("driver", "stub", "pilote de filtrage (stub, windows-wfp, android-vpn, linux-nftables)")
		port       = flag.Int("port", 8443, "port d'écoute du panneau de contrôle (loopback uniquement)")
		dataDir    = flag.String("data", defaultDataDir(), "répertoire de données")
		listOnly   = flag.Bool("list-drivers", false, "affiche les pilotes compilés dans ce binaire et quitte")
	)
	flag.Parse()

	if *listOnly {
		fmt.Println("Pilotes disponibles dans ce binaire:")
		for _, d := range core.AvailableDrivers() {
			fmt.Println("  - " + d)
		}
		return
	}

	// 1. Persistance.
	storePath := filepath.Join(*dataDir, "policies.json")
	store := core.NewFileStore(storePath)

	// 2. Moteur.
	engine := core.NewEngine(store)
	if err := engine.Load(); err != nil {
		log.Printf("[warn] impossible de charger les règles (%v), démarrage à vide", err)
	}

	// 3. Driver natif.
	driver, err := core.NewDriver(core.DriverConfig{
		Driver: *driverName,
	})
	if err != nil {
		log.Fatalf("[fatal] %v", err)
	}
	log.Printf("[main] pilote actif: %s", driver.Name())

	// 4. Démarrage du backend de filtrage (simulé ou réel).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := driver.Start(ctx, engine); err != nil {
			log.Printf("[error] driver.Start: %v", err)
		}
	}()

	// 5. Serveur de contrôle + UI web (loopback).
	server := api.New(engine, driver, *port)

	// 6. Arrêt propre sur Ctrl+C / fermeture.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[main] arrêt demandé...")
		_ = driver.Stop()
		_ = server.Shutdown(context.Background())
		cancel()
	}()

	log.Printf("Pare-feu Faillefox prêt. Panneau: http://127.0.0.1:%d", *port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[fatal] serveur: %v", err)
	}
}

// defaultDataDir renvoie un répertoire de données par plateforme.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".faillefox")
}
