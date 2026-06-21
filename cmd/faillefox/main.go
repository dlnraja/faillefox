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
	"time"

	"github.com/dlnraja/faillefox/internal/api"
	"github.com/dlnraja/faillefox/internal/clamscan"
	"github.com/dlnraja/faillefox/internal/core"
	"github.com/dlnraja/faillefox/internal/correlate"
	"github.com/dlnraja/faillefox/internal/cvefeed"
	"github.com/dlnraja/faillefox/internal/dnsshield"
	_ "github.com/dlnraja/faillefox/internal/drivers/netfw"    // registre windows-netfw
	_ "github.com/dlnraja/faillefox/internal/drivers/nftables" // registre linux-nftables
	_ "github.com/dlnraja/faillefox/internal/drivers/stub"     // registre stub (défaut)
	"github.com/dlnraja/faillefox/internal/freshclam"
	"github.com/dlnraja/faillefox/internal/gamification"
	"github.com/dlnraja/faillefox/internal/logging"
	"github.com/dlnraja/faillefox/internal/platform/winsvc"
	"github.com/dlnraja/faillefox/internal/securitycenter"
	"github.com/dlnraja/faillefox/internal/threatintel"
	"github.com/dlnraja/faillefox/internal/updater"
	"github.com/dlnraja/faillefox/internal/yarascan"
)

func main() {
	var (
		driverName   = flag.String("driver", defaultDriver(), "pilote de filtrage (stub, windows-netfw, linux-nftables)")
		port         = flag.Int("port", 8443, "port d'écoute du panneau de contrôle (loopback uniquement)")
		dataDir      = flag.String("data", defaultDataDir(), "répertoire de données")
		listOnly     = flag.Bool("list-drivers", false, "affiche les pilotes compilés dans ce binaire et quitte")
		profile      = flag.String("profile", "home", "profil réseau (home, office, public)")
		blocklistArg = flag.String("blocklist", "", "fichier hosts à charger comme liste anti-trackers")
		noLog = flag.Bool("no-persistent-log", false, "désactive le journal persistant sur disque")

		// --- v0.3 : bouclier réseau/DNS + CVE + ClamAV ---
		dnsEnabled = flag.Bool("dns", false, "active le résolveur DNS sinkhole (bloque pubs/trackers/malwares au niveau DNS)")
		dnsPort    = flag.Int("dns-port", 5353, "port du résolveur DNS local (loopback)")
		cveEnabled = flag.Bool("cve", false, "active la veille CVE (alerte sur logiciels installés vulnérables)")
		clamscanOn = flag.Bool("clamav", false, "active le scanner ClamAV (nécessite ClamAV installé)")

		// --- v0.4 : automatisation autonome ---
		// auto-update est ACTIVÉ PAR DÉFAUT : le démon télécharge les listes
		// au démarrage puis toutes les 6h. -no-autoupdate pour désactiver.
		noAutoUpdate = flag.Bool("no-autoupdate", false, "désactive l'auto-update des listes DNS/CVE (activé par défaut)")
		updateEvery  = flag.Duration("update-every", 6*time.Hour, "intervalle entre deux mises à jour (défaut 6h)")
		freshclamOn  = flag.Bool("freshclam", false, "active la mise à jour auto des signatures ClamAV (2h)")

		// --- v0.6 : intégration plateforme (service Windows natif) ---
		winsvcCmd = flag.String("winsvc", "", "gestion du service Windows: install|uninstall|start|stop|run")

		// --- v0.7 : threat intel + corrélation + YARA + gamification ---
		threatIntelOn = flag.Bool("threat-intel", false, "agrège les IOC publics (Abuse.ch, OTX, MISP) et les croise")
		yaraRulesArg  = flag.String("yara-rules", "", "fichier de règles YARA publiques à charger (scan simplifié)")
		gameOn        = flag.Bool("gamification", true, "active la gamification (points, badges, streak) — activé par défaut")
	)
	flag.Parse()

	// Gestion du service Windows : -winsvc install|uninstall|start|stop.
	// Bifurcation avant toute autre init.
	switch *winsvcCmd {
	case "install":
		exe, _ := os.Executable()
		if err := winsvc.Install(exe); err != nil {
			log.Fatalf("[fatal] install service: %v", err)
		}
		return
	case "uninstall":
		if err := winsvc.Uninstall(); err != nil {
			log.Fatalf("[fatal] uninstall service: %v", err)
		}
		return
	case "start":
		if err := winsvc.Start(); err != nil {
			log.Fatalf("[fatal] start service: %v", err)
		}
		return
	case "stop":
		if err := winsvc.Stop(); err != nil {
			log.Fatalf("[fatal] stop service: %v", err)
		}
		return
	case "":
		// pas de gestion de service : on continue le démarrage normal
	default:
		log.Fatalf("[fatal] -winsvc valeur inconnue: %s (install|uninstall|start|stop|run)", *winsvcCmd)
	}

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

	// 5b. Auto-update des listes DNS + CVE. ACTIVÉ PAR DÉFAUT en v0.4 : le
	//     démon télécharge les listes (StevenBlack, OISD) au démarrage puis
	//     rafraîchit toutes les `updateEvery` (6h par défaut). Le fetch est
	//     non bloquant (goroutine dédiée), le démon répond immédiatement.
	var upd *updater.Updater
	if !*noAutoUpdate {
		upd = updater.New(bl)
		upd.SetUpdateEvery(*updateEvery)
		go upd.Start(context.Background())
		log.Printf("[main] auto-update activé: sources publiques DNS, rafraîchi toutes les %s", *updateEvery)
	} else {
		log.Printf("[main] auto-update DÉSACTIVÉ (-no-autoupdate)")
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
	var feed *cvefeed.Feed
	if *cveEnabled {
		feed = cvefeed.New()
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

	// 5f. Mise à jour automatique des signatures ClamAV (freshclam). Nécessite
	//     que ClamAV soit installé. Invoque freshclam toutes les 2h pour rester
	//     à jour sur les malwares émergents.
	if *freshclamOn {
		fc := freshclam.New()
		if fc.IsAvailable() {
			go fc.Start(context.Background())
			log.Printf("[main] mise à jour auto des signatures ClamAV activée (toutes les 2h)")
		} else {
			log.Printf("[warn] -freshclam: freshclam non trouvé (installez ClamAV)")
		}
	}

	// 5g. Threat intelligence : agrège les IOC publics (Abuse.ch, OTX, MISP)
	//     en arrière-plan, pour que le corrélateur puisse les croiser.
	var aggregator *threatintel.Aggregator
	var correlator *correlate.Correlator
	if *threatIntelOn {
		aggregator = threatintel.New()
		go func() {
			if n, err := aggregator.FetchAll(context.Background()); err != nil {
				log.Printf("[warn] threat intel: %v", err)
			} else {
				log.Printf("[main] threat intel: %d IOC agrégés", n)
			}
		}()
		correlator = correlate.New(aggregator)
		correlator.SetProfile(p)
		log.Printf("[main] threat intel activé (Abuse.ch + AlienVault OTX, fetch auto)")
	}

	// 5h. Scanner YARA : charge des règles publiques (chargement uniquement,
	//     on NE GÉNÈRE PAS de règles maison). Optionnel.
	var yaraScanner *yarascan.Scanner
	if *yaraRulesArg != "" {
		yaraScanner = yarascan.New()
		if n, err := yaraScanner.LoadRules(*yaraRulesArg); err != nil {
			log.Printf("[warn] YARA rules: %v", err)
		} else {
			log.Printf("[main] scanner YARA activé: %d règle(s) chargée(s) depuis %s", n, *yaraRulesArg)
		}
	}

	// 5i. Gamification : points, badges, streak — encourage la consultation
	//     régulière du panneau. Activée par défaut.
	var game *gamification.State
	if *gameOn {
		gamePath := filepath.Join(*dataDir, "gamification.json")
		game, _ = gamification.New(gamePath)
		newBadges := game.Record(gamification.ActionVisit)
		if err := game.Save(); err != nil {
			log.Printf("[warn] gamification: %v", err)
		}
		if len(newBadges) > 0 {
			log.Printf("[main] gamification: nouveaux badges: %v", newBadges)
		}
		log.Printf("[main] gamification activée: niveau %d, %d points, streak %d j",
			game.Level(), game.Points, game.Streak)
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
	// Branchements optionnels des modules v0.3/v0.4 (nil-safe côté API).
	server.SetUpdater(upd)
	server.SetFeed(feed)
	server.SetScanner(av)

	// 9b. Centre de sécurité v0.9 : vue unifiée de toutes les protections.
	//     On déclare le statut de chaque couche selon ce qui est actif.
	secCenter := securitycenter.New()
	secCenter.SetStatus(securitycenter.ProtFirewall, securitycenter.StatusActive)
	switch driver.Name() {
	case "stub":
		secCenter.SetStatus(securitycenter.ProtFirewall, securitycenter.StatusLimited)
	}
	if dnsShield != nil {
		secCenter.SetStatus(securitycenter.ProtDNS, securitycenter.StatusActive)
		// Le DNS sinkhole active toutes les sous-catégories (pubs/trackers/malware...).
		for _, p := range []securitycenter.Protection{
			securitycenter.ProtAntiAds, securitycenter.ProtAntiTrackers,
			securitycenter.ProtAntiMalware, securitycenter.ProtAntiAdware,
			securitycenter.ProtAntiPhishing,
		} {
			secCenter.SetStatus(p, securitycenter.StatusActive)
		}
		secCenter.SetStats(securitycenter.ProtDNS, map[string]int{
			"domaines_bloques": bl.Size(),
		})
	}
	if av != nil {
		status := securitycenter.StatusActive
		if !av.IsAvailable() {
			status = securitycenter.StatusInactive
		}
		secCenter.SetStatus(securitycenter.ProtAVScanner, status)
	}
	if yaraScanner != nil && yaraScanner.IsAvailable() {
		secCenter.SetStatus(securitycenter.ProtYARAScanner, securitycenter.StatusActive)
	}
	if feed != nil {
		secCenter.SetStatus(securitycenter.ProtCVEFeed, securitycenter.StatusActive)
	}
	if aggregator != nil {
		secCenter.SetStatus(securitycenter.ProtThreatIntel, securitycenter.StatusActive)
	}
	if upd != nil {
		secCenter.SetStatus(securitycenter.ProtAutoUpdate, securitycenter.StatusActive)
	}
	if *freshclamOn {
		secCenter.SetStatus(securitycenter.ProtFreshclam, securitycenter.StatusActive)
	}
	server.SetSecurityCenter(secCenter)
	summary := secCenter.GetSummary()
	log.Printf("[main] centre de sécurité: %d/%d protections actives (score %d%%)",
		summary.Active, summary.Total, summary.Score)

	// 10. Mode service Windows : si on a été lancé par le SCM, on entre en
	//     mode service. Tout le pipeline ci-dessus est initialisé ; on délègue
	//     maintenant à winsvc.Run qui bloquera jusqu'à l'arrêt demandé par le
	//     SCM. Hors Windows ou en exécution normale, on reste en mode console.
	if winsvc.IsWindowsService() {
		log.Printf("[main] démarrage en mode service Windows (SCM)")
		if err := winsvc.Run(func() error {
			return runServer(ctx, server, driver, cancel)
		}); err != nil {
			log.Fatalf("[fatal] service Windows: %v", err)
		}
		return
	}

	// 11. Arrêt propre sur Ctrl+C / fermeture (mode console).
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
	if err := runServer(ctx, server, driver, cancel); err != nil {
		log.Fatalf("[fatal] serveur: %v", err)
	}
}

// runServer lance le serveur HTTP du panneau (loopback). Factorisé pour être
// réutilisé à la fois par le mode console et le mode service Windows.
func runServer(ctx context.Context, server *api.Server, driver core.Driver, cancel context.CancelFunc) error {
	defer cancel()
	return server.ListenAndServe()
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
