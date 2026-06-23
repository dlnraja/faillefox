// Package avsubmit soumet un binaire aux laboratoires antivirus qui
// exposent une API publique gratuite d'upload. Cela augmente la "réputation"
// du binaire : plus il est analysé sans détection, moins les AV le signalent.
//
// Labs supportés (API gratuite, pas de navigateur, pas de captcha) :
//
//   - VirusTotal        (https://www.virustotal.com)   clé: VIRUSTOTAL_API_KEY
//   - Hybrid Analysis   (https://www.hybrid-analysis.com) clé: HYBRID_ANALYSIS_API_KEY
//   - MetaDefender Cloud (https://metadefender.opswat.com) clé: METADEFENDER_API_KEY
//   - MalwareBazaar     (https://bazaar.abuse.ch)       anonyme (pas de clé)
//
// Labs qui n'ont PAS d'API publique (formulaire web only, captcha) :
//   Microsoft Defender, Bitdefender, ESET, Avast, Sophos, Trend Micro...
//   Pour ceux-là, utiliser submit-to-av.ps1 (ouvre les formulaires).
//
// Lancement :
//
//	# Soumettre un fichier aux labs configurés (clés via env)
//	avsubmit -file faillefox.exe
//
//	# Vérifier le statut d'un hash déjà soumis
//	avsubmit -check <sha256>
package avsubmit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// Lab identifie un laboratoire antivirus.
type Lab string

const (
	LabVirusTotal     Lab = "virustotal"
	LabHybridAnalysis Lab = "hybrid-analysis"
	LabMetaDefender   Lab = "metadefender"
	LabMalwareBazaar  Lab = "malwarebazaar"
)

// Config contient les clés API pour chaque lab (vides = lab non configuré).
type Config struct {
	VirusTotalKey     string
	HybridAnalysisKey string
	MetaDefenderKey   string
}

// Result est le résultat d'une soumission à un lab.
type Result struct {
	Lab        Lab    `json:"lab"`
	Success    bool   `json:"success"`
	Submission string `json:"submission_id"` // ID retourné par le lab
	URL        string `json:"url"`           // lien d'analyse (si dispo)
	Error      string `json:"error,omitempty"`
}

// Submit soumet un fichier à tous les labs configurés. Renvoie un résultat
// par lab. Les labs sans clé API sont ignorés (pas d'erreur).
func Submit(filePath string, cfg Config) []Result {
	var results []Result
	if cfg.VirusTotalKey != "" {
		results = append(results, submitVirusTotal(filePath, cfg.VirusTotalKey))
	} else {
		results = append(results, Result{Lab: LabVirusTotal, Error: "clé API non configurée"})
	}
	if cfg.HybridAnalysisKey != "" {
		results = append(results, submitHybridAnalysis(filePath, cfg.HybridAnalysisKey))
	} else {
		results = append(results, Result{Lab: LabHybridAnalysis, Error: "clé API non configurée"})
	}
	if cfg.MetaDefenderKey != "" {
		results = append(results, submitMetaDefender(filePath, cfg.MetaDefenderKey))
	} else {
		results = append(results, Result{Lab: LabMetaDefender, Error: "clé API non configurée"})
	}
	// MalwareBazaar ne nécessite pas de clé (mais n'accepte que les malwares,
	// pas les faux positifs — donc on ne soumet pas ici, c'est juste listé).
	results = append(results, Result{Lab: LabMalwareBazaar, Error: "non applicable (malwares seulement)"})
	return results
}

// submitVirusTotal upload un fichier à VirusTotal via l'API v3.
// Limite gratuite : 4 requêtes/min, 500 MB max par fichier.
func submitVirusTotal(filePath, apiKey string) Result {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := os.Open(filePath)
	if err != nil {
		return Result{Lab: LabVirusTotal, Error: err.Error()}
	}
	defer func() { _ = file.Close() }()
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return Result{Lab: LabVirusTotal, Error: err.Error()}
	}
	if _, err := io.Copy(part, file); err != nil {
		return Result{Lab: LabVirusTotal, Error: err.Error()}
	}
	_ = writer.Close()

	req, _ := http.NewRequest("POST", "https://www.virustotal.com/api/v3/files", body)
	req.Header.Set("x-apikey", apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Lab: LabVirusTotal, Error: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Result{Lab: LabVirusTotal, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	var data struct {
		Data struct {
			ID  string `json:"id"`
			Links struct {
				Self string `json:"self"`
			} `json:"links"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Result{Lab: LabVirusTotal, Success: true, Submission: "?", Error: "réponse illisible mais uploadé"}
	}
	return Result{
		Lab:        LabVirusTotal,
		Success:    true,
		Submission: data.Data.ID,
		URL:        "https://www.virustotal.com/gui/file/" + data.Data.ID,
	}
}

// submitHybridAnalysis upload à Hybrid Analysis (FalconCrowdStrike).
// API v2 : https://hybrid-analysis.com/docs/api/v2
func submitHybridAnalysis(filePath, apiKey string) Result {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := os.Open(filePath)
	if err != nil {
		return Result{Lab: LabHybridAnalysis, Error: err.Error()}
	}
	defer func() { _ = file.Close() }()
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return Result{Lab: LabHybridAnalysis, Error: err.Error()}
	}
	if _, err := io.Copy(part, file); err != nil {
		return Result{Lab: LabHybridAnalysis, Error: err.Error()}
	}
	_ = writer.WriteField("environment_id", "110") // Windows 10 x64
	_ = writer.WriteField("hybrid_analysis_type", "re_scan")
	_ = writer.Close()

	req, _ := http.NewRequest("POST", "https://www.hybrid-analysis.com/api/v2/submit/file", body)
	req.Header.Set("api-key", apiKey)
	req.Header.Set("User-Agent", "Falcon Sandbox")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Lab: LabHybridAnalysis, Error: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return Result{Lab: LabHybridAnalysis, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return Result{
		Lab:        LabHybridAnalysis,
		Success:    true,
		Submission: "submitted",
		URL:        "https://www.hybrid-analysis.com/search?query=" + sha256hex(filePath),
	}
}

// submitMetaDefender upload à MetaDefender Cloud (OPSWAT).
// API v4 : https://products.opswat.com/mdcloud/metadefender-cloud-core-api-v4.html
func submitMetaDefender(filePath, apiKey string) Result {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return Result{Lab: LabMetaDefender, Error: err.Error()}
	}
	req, _ := http.NewRequest("POST", "https://api.metadefender.com/v4/file", bytes.NewReader(file))
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("filename", filePath)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Lab: LabMetaDefender, Error: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Result{Lab: LabMetaDefender, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	var data struct {
		DataID string `json:"data_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Result{Lab: LabMetaDefender, Success: true, Submission: "?", Error: "réponse illisible mais uploadé"}
	}
	return Result{
		Lab:        LabMetaDefender,
		Success:    true,
		Submission: data.DataID,
		URL:        "https://metadefender.opswat.com/results#!/file/" + data.DataID + "/regular/overview",
	}
}

// CheckHash vérifie si un hash SHA256 est déjà connu des labs (sans upload).
// Utile pour voir les résultats d'une soumission précédente.
func CheckHash(sha256Hash string, cfg Config) map[Lab]string {
	results := make(map[Lab]string)
	if cfg.VirusTotalKey != "" {
		results[LabVirusTotal] = "https://www.virustotal.com/api/v3/files/" + sha256Hash
	}
	if cfg.HybridAnalysisKey != "" {
		results[LabHybridAnalysis] = "https://www.hybrid-analysis.com/api/v2/search/hash"
	}
	if cfg.MetaDefenderKey != "" {
		results[LabMetaDefender] = "https://api.metadefender.com/v4/hash/" + sha256Hash
	}
	return results
}

// sha256hex calcule le SHA256 d'un fichier (helper).
func sha256hex(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
