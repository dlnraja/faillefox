// Package setup fournit un wizard de configuration interactif qui valide
// en une seule commande tous les prérequis de Faillefox :
//
//   - Présence du token d'auth
//   - Clés API antivirus (VirusTotal, Hybrid Analysis, MetaDefender)
//   - Configuration SignPath
//   - Signature ClamAV
//   - Compilation
//
// Lancement : setup (binaire dédié cmd/setup)
package setup

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// CheckResult est le résultat d'une vérification.
type CheckResult struct {
	Name     string
	Status   string // "ok", "warn", "fail", "skip"
	Detail   string
}

// RunAll exécute toutes les vérifications et renvoie un rapport.
func RunAll(dataDir string) []CheckResult {
	var results []CheckResult

	// 1. Token d'authentification.
	results = append(results, checkToken(dataDir))

	// 2. Clés API antivirus.
	results = append(results, checkEnvKey("VIRUSTOTAL_API_KEY", "VirusTotal API"))
	results = append(results, checkEnvKey("HYBRID_ANALYSIS_API_KEY", "Hybrid Analysis API"))
	results = append(results, checkEnvKey("METADEFENDER_API_KEY", "MetaDefender API"))

	// 3. SignPath.
	results = append(results, checkEnvKey("SIGNPATH_API_TOKEN", "SignPath (signature)"))

	// 4. Freshclam (signatures ClamAV).
	results = append(results, checkFreshclam())

	// 5. Connectivité loopback (le démon tourne-t-il ?).
	results = append(results, checkLoopback())

	return results
}

// checkToken vérifie la présence du token d'auth.
func checkToken(dataDir string) CheckResult {
	path := dataDir + "/token"
	if _, err := os.Stat(path); err != nil {
		return CheckResult{Name: "Token d'auth", Status: "warn",
			Detail: "non généré (le sera au premier démarrage du démon)"}
	}
	return CheckResult{Name: "Token d'auth", Status: "ok",
		Detail: "présent dans " + path}
}

// checkEnvKey vérifie la présence d'une variable d'environnement.
func checkEnvKey(envVar, name string) CheckResult {
	val := os.Getenv(envVar)
	if val == "" {
		return CheckResult{Name: name, Status: "skip",
			Detail: "non configuré (" + envVar + " vide) — optionnel"}
	}
	if len(val) < 10 {
		return CheckResult{Name: name, Status: "warn",
			Detail: "clé suspecte (trop courte)"}
	}
	return CheckResult{Name: name, Status: "ok",
		Detail: "configuré (" + envVar + ")"}
}

// checkFreshclam vérifie si freshclam/ClamAV est installé.
func checkFreshclam() CheckResult {
	_, err := exeLookPath("freshclam")
	if err != nil {
		return CheckResult{Name: "ClamAV (freshclam)", Status: "skip",
			Detail: "non installé — scanner ClamAV désactivé (voir docs/clamav.md)"}
	}
	return CheckResult{Name: "ClamAV (freshclam)", Status: "ok",
		Detail: "installé"}
}

// checkLoopback vérifie si le démon Faillefox répond sur 127.0.0.1:8443.
func checkLoopback() CheckResult {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8443/api/status")
	if err != nil {
		return CheckResult{Name: "Démon (loopback)", Status: "warn",
			Detail: "non joignable sur 127.0.0.1:8443 — lancez 'faillefox' d'abord"}
	}
	defer func() { _ = resp.Body.Close() }()
	return CheckResult{Name: "Démon (loopback)", Status: "ok",
		Detail: fmt.Sprintf("répond (HTTP %d)", resp.StatusCode)}
}

// Print affiche le rapport de manière lisible.
func Print(results []CheckResult) {
	fmt.Println("\n╔══════════════════════════════════════════════════════╗")
	fmt.Println("║       Configuration Faillefox — vérification          ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	ok, warn, fail, skip := 0, 0, 0, 0
	for _, r := range results {
		var icon, color string
		switch r.Status {
		case "ok":
			icon, color = "✅", "\033[32m"
		case "warn":
			icon, color = "⚠️ ", "\033[33m"
		case "fail":
			icon, color = "❌", "\033[31m"
		case "skip":
			icon, color = "⏭️ ", "\033[2m"
		}
		fmt.Printf("  %s %s%-30s\033[0m %s\n", icon, color, r.Name, r.Detail)
		switch r.Status {
		case "ok":
			ok++
		case "warn":
			warn++
		case "fail":
			fail++
		case "skip":
			skip++
		}
	}
	fmt.Printf("\n  Résumé : %d OK, %d warnings, %d échecs, %d optionnels skip\n\n", ok, warn, fail, skip)
	if fail > 0 {
		fmt.Println("  ❌ Des vérifications ont échoué. Corrigez-les avant de continuer.")
	} else if warn > 0 {
		fmt.Println("  ⚠ Certains éléments sont optionnels. Voir les détails ci-dessus.")
	} else {
		fmt.Println("  ✅ Tout est configuré ! Faillefox est prêt.")
	}
}
