// Command faillefox est le point d'entrée du pare-feu Faillefox.
//
// Lancement:
//
//	faillefox                          # driver stub + UI sur http://127.0.0.1:8443
//	faillefox -driver windows-netfw    # Pare-feu Windows réel (droits admin)
//	faillefox -driver linux-nftables   # nftables/iptables (root)
//	faillefox -port 9000
//	faillefox -list-drivers            # affiche les pilotes compilés
//	faillefox -profile public          # profil réseau (home/office/public)
//	faillefox -blocklist data/blocklist.txt  # charge une liste anti-trackers
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/dlnraja/faillefox/internal/api"
	"github.com/dlnraja/faillefox/internal/clamscan"
	"github.com/dlnraja/faillefox/internal/core"
	"github.com/dlnraja/faillefox/internal/cvefeed"
	"github.com/dlnraja/faillefox/internal/dnsshield"
	_ "github.com/dlnraja/faillefox/internal/drivers/netfw"    // registre windows-netfw
	_ "github.com/dlnraja/faillefox/internal/drivers/nftables" // registre linux-nftables
	_ "github.com/dlnraja/faillefox/internal/drivers/stub"     // registre stub (défaut)
	"github.com/dlnraja/faillefox/internal/logging"
	"github.com/dlnraja/faillefox/internal/updater"
)

func main() {
	var (
		driverName   = flag.String("driver", defaultDriver(), "pilote de filtrage (stub, windows-netfw, linux-nftables)")
		port         = flag.Int("port", 8443, "port d'écoute du panneau de contrôle (loopback uniquement)")
		dataDir      = flag.String("data", defaultDataDir(), "répertoire de données")
		listOnly     = flag.Bool("list-drivers", false, "affiche les pilotes compilés dans ce binaire et quitte")
		profile      = flag.String("profile", "home", "profil réseau (home, office, public)")
		blocklistArg = flag.String("blocklist", "", "fichier hosts à charger comme liste anti-trackers")
		noLog        = flag.Bool("no-persistent-log", false, "désactive le journal persistant sur disque")

		// --- v0.3 : bouclier réseau/DNS + CVE + ClamAV ---
		dnsEnabled   = flag.Bool("dns", false, "active le résolveur DNS sinkhole (bloque pubs/trackers/malwares au niveau DNS)")
		dnsPort      = flag.Int("dns-port", 5353, "port du résolveur DNS local (loopback)")
		autoUpdate   = flag.Bool("auto-update", false, "télécharge/rafraîchit automatiquement les listes DNS et CVE (24h)")
		cveEnabled   = flag.Bool("cve", false, "active la veille CVE (alerte sur logiciels installés vulnérables)")
		clamscanOn   = flag.Bool("clamav", false, "active le scanner ClamAV (nécessite ClamAV installé)")
	)
	flag.Parse()

	if *listOnly {
		fmt.Println("Pilotes disponibles dans ce binaire:")
		for _, d := range core.AvailableDrivers() {
			fmt.Println("  - " + d)
		}
		return
	}

	// 1. Persistance des règles.
	storePath := filepath.Join(*dataDir, "policies.json")
	store := core.NewFileStore(storePath)

	// 2. Moteur + chargement des règles existantes.
	engine := core.NewEngine(store)
	if err := engine.Load(); err != nil {
		log.Printf("[warn] impossible de charger les règles (%v), démarrage à vide", err)
	}

	// 3. Journal persistant rotatif (sauf si désactivé).
	if !*noLog {
		logger, err := logging.NewRotatingLogger(*dataDir, "events")
		if err != nil {
			log.Printf("[warn] journal persistant indisponible: %v", err)
		} else {
			engine.SetSink(logger.Write)
			defer func() { _ = logger.Close() }()
			log.Printf("[main] journal persistant activé: %s", filepath.Join(*dataDir, "events.jsonl"))
		}
	}

	// 4. Blocklist anti-trackers (partagée entre moteur et DNS sinkhole).
	//     Soit chargée depuis un fichier local, soit (auto-update) téléchargée
	//     depuis des listes publiques (StevenBlack, OISD).
	bl := core.NewBlocklist()
	if *blocklistArg != "" {
		data, err := os.ReadFile(*blocklistArg)
		if err != nil {
			log.Printf("[warn] blocklist illisible (%v)", err)
		} else {
			n := bl.LoadFromHosts(string(data))
			log.Printf("[main] blocklist chargée: %d domaine(s)", n)
		}
	}
	engine.SetBlocklist(bl)

	// 5. Profil réseau (détermine la politique par défaut conseillée).
	p := core.Profile(*profile)
	pm := core.NewProfileManager(p)
	log.Printf("[main] profil réseau: %s (défaut conseillé: %s)", p, core.DefaultForProfile(p))
	pm.OnChange(func(newProfile core.Profile) {
		log.Printf("[main] profil changé -> %s", newProfile)
	})

	// 5b. Auto-update des listes DNS (télécharge StevenBlack + OISD dans la
	//     blocklist partagée). Optionnel, désactivé par défaut car il fait
	//     des appels réseau au démarrage.
	if *autoUpdate {
		upd := updater.New(bl)
		go upd.Start(context.Background())
		log.Printf("[main] auto-update des listes DNS activé (sources publiques)")
	}

	// 5c. Résolveur DNS sinkhole (bloque pubs/trackers/malwares au niveau DNS,
	//     façon Pi-hole local). Optionnel. Écoute sur 127.0.0.1.
	var dnsShield *dnsshield.Shield
	if *dnsEnabled {
		dnsShield = dnsshield.New(*dnsPort)
		dnsShield.SetBlocklist(bl)
		go func() {
			if err := dnsShield.Start(context.Background()); err != nil {
				log.Printf("[error] DNS sinkhole: %v", err)
			}
		}()
		log.Printf("[main] DNS sinkhole activé: 127.0.0.1:%d (configurez votre OS pour l'utiliser)", *dnsPort)
	}

	// 5d. Veille CVE : interroge la base NVD (gratuite, officielle) et alerte
	//     si un logiciel installé a une vulnérabilité connue.
	if *cveEnabled {
		feed := cvefeed.New()
		go func() {
			if err := feed.RefreshAll(context.Background()); err != nil {
				log.Printf("[warn] veille CVE: %v", err)
			}
		}()
		log.Printf("[main] veille CVE activée (base NVD officielle, gratuite)")
	}

	// 5e. Scanner ClamAV : seul moteur AV open source, mais LIMITÉ vs les
	//     solutions commerciales. Ne remplace pas un AV temps réel.
	var av *clamscan.Scanner
	if *clamscanOn {
		av = clamscan.New()
		av.LogAvailability()
	}

	// 6. Driver natif.
	driver, err := core.NewDriver(core.DriverConfig{
		Driver: *driverName,
	})
	if err != nil {
		log.Fatalf("[fatal] %v", err)
	}
	log.Printf("[main] pilote actif: %s", driver.Name())

	// 7. Application initiale des règles au backend natif.
	if err := driver.ApplyRules(engine.Rules()); err != nil {
		log.Printf("[warn]ApplyRules initial: %v", err)
	}

	// 8. Démarrage du backend (boucle d'interception ou no-op selon le pilote).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := driver.Start(ctx, engine); err != nil {
			log.Printf("[error] driver.Start: %v", err)
		}
	}()

	// 9. Serveur de contrôle + UI web (loopback).
	server := api.New(engine, driver, *port)

	// 10. Arrêt propre sur Ctrl+C / fermeture.
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

// defaultDriver renvoie le pilote par défaut selon l'OS :
// - Windows : windows-netfw (Pare-feu Windows réel)
// - Linux   : linux-nftables (nftables/iptables)
// - autres  : stub (simulation)
func defaultDriver() string {
	switch runtimeOS() {
	case "windows":
		return "windows-netfw"
	case "linux":
		return "linux-nftables"
	default:
		return "stub"
	}
}

// runtimeOS renvoie le GOOS courant (windows/linux/darwin).
func runtimeOS() string {
	return runtime.GOOS
}

// defaultDataDir renvoie un répertoire de données par plateforme.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".faillefox")
}
