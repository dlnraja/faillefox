// Package clamscan intègre le moteur ClamAV — le SEUL moteur antivirus open
// source largement déployé et maintenu. On l'utilise en deux modes :
//
//   1. clamd (daemon) : si ClamAV est installé et son daemon tourne, on
//      communique avec lui via son protocole (port 3310) — scan rapide,
//      idéal pour scanner en masse.
//   2. clamscan (CLI)  : sinon, on invoque le binaire clamscan en ligne de
//      commande pour un scan ponctuel.
//
// IMPORTANT — HONNÊTETÉ SUR LES LIMITES :
//   ClamAV est un moteur antivirus LIBRE et gratuit, mais il est NETTEMENT
//   INFERIEUR aux solutions commerciales (Kaspersky, Bitdefender, ESET) :
//     - Pas d'heuristique ML avancée (détection comportementale limitée)
//     - Pas de sandbox intégrée (analyse en bac à sable)
//     - Détection essentiellement basée sur signatures (connues)
//     - Taux de détection sur malwares zero-day: faible vs solutions pro
//
//   ClamAV est utile pour :
//     - Scanner des fichiers téléchargés avant exécution
//     - Vérifier des clés USB / archives
//     - Détecter des malwares CONNUS dans une collection de fichiers
//
//   ClamAV NE REMPLACE PAS une véritable solution antivirus temps réel.
//
// Installation (à documenter dans docs/clamav.md) :
//   Windows : https://www.clamav.net/downloads (installer officiel)
//   Linux   : apt install clamav clamav-daemon ; systemctl start clamav-daemon
//   macOS   : brew install clamav
package clamscan

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Scanner encapsule l'accès à ClamAV (daemon si dispo, sinon CLI).
type Scanner struct {
	clamdAddr string // ex: "127.0.0.1:3310"
	timeout   time.Duration
}

// New crée un scanner qui tentera d'abord clamd, puis clamscan en fallback.
func New() *Scanner {
	return &Scanner{
		clamdAddr: "127.0.0.1:3310",
		timeout:   60 * time.Second,
	}
}

// Result est le résultat d'un scan.
type Result struct {
	Path      string `json:"path"`
	Infected  bool   `json:"infected"`
	Signature string `json:"signature,omitempty"` // nom du malware si détecté
	Detail    string `json:"detail,omitempty"`    // message humain
}

// IsAvailable vérifie si ClamAV est installé/accessible (clamd OU clamscan).
func (s *Scanner) IsAvailable() bool {
	return s.clamdRunning() || s.clamscanInstalled()
}

// clamdRunning teste si le daemon ClamAV écoute.
func (s *Scanner) clamdRunning() bool {
	conn, err := net.DialTimeout("tcp", s.clamdAddr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// clamscanInstalled teste si le binaire clamscan est dans le PATH.
func (s *Scanner) clamscanInstalled() bool {
	_, err := exec.LookPath("clamscan")
	return err == nil
}

// ScanFile analyse un fichier. Tente clamd d'abord, sinon clamscan CLI.
func (s *Scanner) ScanFile(ctx context.Context, path string) (Result, error) {
	if s.clamdRunning() {
		return s.scanViaDaemon(ctx, path)
	}
	if s.clamscanInstalled() {
		return s.scanViaCLI(ctx, path)
	}
	return Result{}, fmt.Errorf("ClamAV non disponible (installez clamd ou clamscan)")
}

// scanViaDaemon utilise le protocole clamd : "SCAN <path>\n".
func (s *Scanner) scanViaDaemon(ctx context.Context, path string) (Result, error) {
	d := net.Dialer{Timeout: s.timeout}
	conn, err := d.DialContext(ctx, "tcp", s.clamdAddr)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = conn.Close() }()

	// Commande clamd : "SCAN chemin_absolu".
	if _, err := fmt.Fprintf(conn, "SCAN %s\n", path); err != nil {
		return Result{}, err
	}

	// Réponse typique : "<path>: OK" ou "<path>: <SIGNATURE> FOUND".
	sc := bufio.NewScanner(conn)
	if !sc.Scan() {
		return Result{}, fmt.Errorf("réponse clamd vide")
	}
	return parseClamdLine(path, sc.Text()), nil
}

// scanViaCLI invoque clamscan en ligne de commande.
// Sortie typique : "/path/file: OK." ou "/path/file: Eicar.Test.Signature FOUND".
func (s *Scanner) scanViaCLI(ctx context.Context, path string) (Result, error) {
	cmd := exec.CommandContext(ctx, "clamscan", "--no-summary", "--infected", path)
	out, err := cmd.Output()
	// clamscan renvoie exit code 1 si un malware est trouvé — ce n'est PAS une erreur.
	output := string(out)
	if err != nil {
		// Exit code 1 = infection détectée (pas une vraie erreur).
		if strings.Contains(output, "FOUND") {
			return parseClamscanOutput(path, output), nil
		}
		return Result{}, fmt.Errorf("clamscan: %w (%s)", err, output)
	}
	return parseClamscanOutput(path, output), nil
}

// parseClamdLine interprète une ligne de réponse clamd.
// Format : "<path>: OK"  ou  "<path>: <SIGNATURE> FOUND"
func parseClamdLine(path, line string) Result {
	r := Result{Path: path}
	if strings.Contains(line, "FOUND") {
		r.Infected = true
		// Extraction de la signature entre ": " et " FOUND".
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			r.Signature = strings.TrimSuffix(parts[1], " FOUND")
		}
		r.Detail = "Malware détecté: " + r.Signature
	} else if strings.Contains(line, "OK") {
		r.Detail = "Aucune infection connue détectée"
	}
	return r
}

// parseClamscanOutput interprète la sortie de clamscan CLI.
func parseClamscanOutput(path, output string) Result {
	r := Result{Path: path}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "FOUND") {
			r.Infected = true
			// "/path/file: SIGNATURE FOUND"
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				r.Signature = strings.TrimSuffix(parts[1], " FOUND")
			}
			r.Detail = "Malware détecté: " + r.Signature
			break
		}
	}
	if !r.Infected {
		r.Detail = "Aucune infection connue détectée"
	}
	return r
}

// LogAvailability journalise une fois l'état de ClamAV.
func (s *Scanner) LogAvailability() {
	if !s.IsAvailable() {
		log.Printf("[clamav] NON disponible — scan désactivé. " +
			"Installez ClamAV pour activer (voir docs/clamav.md).")
	} else if s.clamdRunning() {
		log.Printf("[clamav] daemon clamd détecté (127.0.0.1:3310)")
	} else {
		log.Printf("[clamav] binaire clamscan détecté (mode CLI)")
	}
}
