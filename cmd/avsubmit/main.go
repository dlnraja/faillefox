// Command avsubmit soumet un binaire aux laboratoires antivirus via leurs
// API publiques gratuites (pas de navigateur, pas de captcha).
//
// Labs supportés :
//   - VirusTotal (clé gratuite sur https://www.virustotal.com)
//   - Hybrid Analysis (clé gratuite sur https://www.hybrid-analysis.com)
//   - MetaDefender Cloud (clé gratuite sur https://metadefender.opswat.com)
//
// Les clés API sont lues depuis les variables d'environnement :
//   VIRUSTOTAL_API_KEY, HYBRID_ANALYSIS_API_KEY, METADEFENDER_API_KEY
//
// Lancement :
//
//	avsubmit -file faillefox.exe
//	avsubmit -check <sha256>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dlnraja/faillefox/internal/avsubmit"
)

func main() {
	var (
		filePath = flag.String("file", "", "fichier à soumettre aux labs AV")
		checkHash = flag.String("check", "", "vérifier un hash SHA256 (sans upload)")
		pretty   = flag.Bool("pretty", true, "affichage JSON indenté")
	)
	flag.Parse()

	if *filePath == "" && *checkHash == "" {
		fmt.Println("avsubmit — soumission automatique aux labs antivirus")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  avsubmit -file faillefox.exe      Soumet le fichier aux labs configurés")
		fmt.Println("  avsubmit -check <sha256>          Vérifie un hash (sans upload)")
		fmt.Println()
		fmt.Println("Variables d'environnement (clés API gratuites) :")
		fmt.Println("  VIRUSTOTAL_API_KEY      https://www.virustotal.com (Profile > API Key)")
		fmt.Println("  HYBRID_ANALYSIS_API_KEY https://www.hybrid-analysis.com (Profile > API Key)")
		fmt.Println("  METADEFENDER_API_KEY    https://metadefender.opswat.com (Sign up > API)")
		fmt.Println()
		fmt.Println("Les labs sans clé sont ignorés (pas d'erreur).")
		os.Exit(0)
	}

	cfg := avsubmit.Config{
		VirusTotalKey:     os.Getenv("VIRUSTOTAL_API_KEY"),
		HybridAnalysisKey: os.Getenv("HYBRID_ANALYSIS_API_KEY"),
		MetaDefenderKey:   os.Getenv("METADEFENDER_API_KEY"),
	}

	if *checkHash != "" {
		// Mode check : juste les URLs d'API pour vérifier un hash.
		urls := avsubmit.CheckHash(*checkHash, cfg)
		for lab, url := range urls {
			fmt.Printf("  %s: %s\n", lab, url)
		}
		return
	}

	// Mode submit : upload le fichier aux labs.
	if _, err := os.Stat(*filePath); err != nil {
		log.Fatalf("[fatal] fichier introuvable: %v", err)
	}

	fmt.Printf("Soumission de %s aux labs AV...\n\n", *filePath)
	results := avsubmit.Submit(*filePath, cfg)

	var enc func(v any) []byte
	if *pretty {
		enc = func(v any) []byte {
			b, _ := json.MarshalIndent(v, "", "  ")
			return b
		}
	} else {
		enc = func(v any) []byte {
			b, _ := json.Marshal(v)
			return b
		}
	}

	fmt.Println(string(enc(results)))
	fmt.Println()

	// Résumé.
	success := 0
	for _, r := range results {
		if r.Success {
			success++
			fmt.Printf("  ✅ %s — %s\n", r.Lab, r.URL)
		} else {
			fmt.Printf("  ⏭️  %s — %s\n", r.Lab, r.Error)
		}
	}
	fmt.Printf("\n%d/%d labs ont accepté la soumission.\n", success, len(results))
}
