//go:build windows

// Package winsvc permet à Faillefox de s'exécuter comme un service Windows
// natif (démarrage automatique au boot, gestion via services.msc, arrêt
// propre, journalisation dans l'Event Viewer).
//
// Utilisation :
//
//	# Installer le service
//	faillefox.exe -winsvc install
//
//	# Démarrer / arrêter
//	faillefox.exe -winsvc start
//	faillefox.exe -winsvc stop
//
//	# Désinstaller
//	faillefox.exe -winsvc uninstall
//
//	# Mode service (lancé par le SCM Windows, pas par l'utilisateur)
//	faillefox.exe -winsvc run
//
// Une fois installé, Faillefox démarre automatiquement au boot de Windows,
// avant même qu'un utilisateur ne se connecte.
package winsvc

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const svcName = "Faillefox"
const svcDesc = "Faillefox — pare-feu libre et multiplateforme (protection réseau, DNS sinkhole, veille CVE, scanner ClamAV)"

// IsWindowsService indique si le processus courant est lancé par le SCM
// Windows (Service Control Manager). Permet au main de bifurquer vers le
// mode service quand c'est le cas.
func IsWindowsService() bool {
	is, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return is
}

// Run exécute Faillefox en tant que service Windows. Bloquant. La fonction
// runFn est appelée dans une goroutine dédiée et reçoit un contexte qui est
// annulé quand Windows demande l'arrêt du service.
func Run(runFn func() error) error {
	return svc.Run(svcName, &service{runFn: runFn})
}

type service struct {
	runFn func() error
}

// Execute est appelé par le SCM. Il gère les commandes start/stop.
func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Lancement effectif de Faillefox en arrière-plan.
	stopCh := make(chan struct{})
	go func() {
		if err := s.runFn(); err != nil {
			log.Printf("[winsvc] erreur: %v", err)
		}
		close(stopCh)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	log.Printf("[winsvc] service %s démarré", svcName)

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Printf("[winsvc] arrêt demandé")
				changes <- svc.Status{State: svc.StopPending}
				break loop
			default:
			}
		case <-stopCh:
			break loop
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}

// Install enregistre Faillefox comme service Windows à démarrage automatique.
// exePath est le chemin absolu du binaire (généralement os.Executable()).
func Install(exePath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connexion SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err == nil {
		_ = s.Close()
		return fmt.Errorf("le service %s existe déjà", svcName)
	}

	s, err = m.CreateService(svcName, exePath, mgr.Config{
		DisplayName: svcName,
		Description: svcDesc,
		StartType:   mgr.StartAutomatic, // démarrage au boot
	})
	if err != nil {
		return fmt.Errorf("création du service: %w", err)
	}
	defer s.Close()

	log.Printf("[winsvc] service %s installé (démarrage automatique)", svcName)
	return nil
}

// Uninstall supprime le service Windows.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connexion SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service introuvable: %w", err)
	}
	defer s.Close()

	// Tente d'arrêter le service d'abord (ignore l'erreur s'il n'est pas démarré).
	_, _ = s.Control(svc.Stop)
	time.Sleep(500 * time.Millisecond)

	if err := s.Delete(); err != nil {
		return fmt.Errorf("suppression du service: %w", err)
	}
	log.Printf("[winsvc] service %s désinstallé", svcName)
	return nil
}

// Start démarre le service (via SCM).
func Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(svcName)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

// Stop demande l'arrêt du service.
func Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(svcName)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}

// status (placeholder pour des extensions futures : afficher l'état du service)
var _ = os.Getpid
