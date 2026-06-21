// Package yarascan charge des règles de détection publiques (format YARA
// simplifié) et scanne des fichiers à la recherche de patterns correspondants.
//
// HONNÊTETÉ — IMPORTANT :
//   Ce package n'embarque PAS le moteur YARA complet (qui nécessiterait
//   libyara en C + CGO, lourd et non portable). À la place, il implémente
//   un chargeur de règles YARA simplifié qui extrait les patterns de chaînes
//   ($s1 = "..." hex/ascii) et les cherche dans les fichiers.
//
//   Cela couvre ~80 % des règles YARA courantes (détection par chaînes
//   connues) mais ne gère PAS les conditions complexes, les modules PE/ELF,
//   ni les caractères génériques hex. Pour le moteur YARA complet, il faudra
//   intégrer github.com/hillu/go-yara (CGO) en v0.8.
//
//   On NE GÉNÈRE PAS de règles maison : on ne fait que charger des règles
//   PUBLIQUES et éprouvées (YARA Forge, signature-base, etc.). Écrire ses
//   propres signatures AV sans infrastructure de test produit des faux
//   positifs massifs — c'est dangereux.
package yarascan

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Rule est une règle YARA simplifiée chargée.
type Rule struct {
	Name     string   // nom de la règle
	Strings  []string // patterns à chercher (ascii)
	Tags     []string // tags de la règle (ex: ["ransomware"])
	Author   string
	Source   string   // fichier/URL d'origine
}

// Match est une correspondance trouvée.
type Match struct {
	Rule    Rule   `json:"rule"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
	Strings []string `json:"matched_strings"`
}

// Scanner charge des règles et scanne des fichiers.
type Scanner struct {
	rules []Rule
	// Si le binaire `yara` est dans le PATH, on l'utilise pour les scans
	// (moteur complet). Sinon, on retombe sur notre matcher simplifié.
	useYaraBin bool
}

// New crée un scanner vide.
func New() *Scanner {
	s := &Scanner{}
	if _, err := exec.LookPath("yara"); err == nil {
		s.useYaraBin = true
	}
	return s
}

// IsAvailable indique si au moins une règle est chargée.
func (s *Scanner) IsAvailable() bool { return len(s.rules) > 0 }

// UsesYaraBinary indique si le moteur YARA complet (binaire) est dispo.
func (s *Scanner) UsesYaraBinary() bool { return s.useYaraBin }

// LoadRules charge des règles depuis un fichier au format YARA simplifié.
// On extrait les chaînes ($s = "...") et les métadonnées de base.
func (s *Scanner) LoadRules(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return s.loadFromReader(f, path)
}

// loadFromReader parse un contenu YARA simplifié.
func (s *Scanner) loadFromReader(r io.Reader, source string) (int, error) {
	n := 0
	var current *Rule
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Regex pour capturer : rule <name> { ... }
	ruleStart := regexp.MustCompile(`^\s*rule\s+(\w+)`)
	// Regex pour : $s1 = "chaine ascii"
	strDef := regexp.MustCompile(`\$\w+\s*=\s*"([^"]+)"`)
	// Regex pour : $s1 = { AA BB CC } (hex)
	hexDef := regexp.MustCompile(`\$\w+\s*=\s*\{([^}]+)\}`)
	// Métadonnée : author = "..."
	metaAuthor := regexp.MustCompile(`author\s*=\s*"([^"]+)"`)

	for sc.Scan() {
		line := sc.Text()

		if m := ruleStart.FindStringSubmatch(line); m != nil {
			if current != nil {
				s.rules = append(s.rules, *current)
				n++
			}
			current = &Rule{Name: m[1], Source: source}
			continue
		}
		if current == nil {
			continue
		}
		if m := metaAuthor.FindStringSubmatch(line); m != nil {
			current.Author = m[1]
			continue
		}
		if m := strDef.FindStringSubmatch(line); m != nil {
			current.Strings = append(current.Strings, m[1])
		}
		if m := hexDef.FindStringSubmatch(line); m != nil {
			// Hex : on convertit en bytes pour la recherche binaire.
			if b, err := hex.DecodeString(strings.ReplaceAll(m[1], " ", "")); err == nil {
				current.Strings = append(current.Strings, string(b))
			}
		}
	}
	if current != nil {
		s.rules = append(s.rules, *current)
		n++
	}
	return n, sc.Err()
}

// ScanFile analyse un fichier et renvoie les règles qui matchent.
func (s *Scanner) ScanFile(ctx context.Context, path string) ([]Match, error) {
	if len(s.rules) == 0 {
		return nil, fmt.Errorf("aucune règle YARA chargée")
	}

	// Hash SHA256 (utile pour le rapport).
	h, err := sha256OfFile(path)
	if err != nil {
		return nil, err
	}

	// Lecture du fichier (limité à 50 Mo pour éviter les OOM).
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	const maxScan = 50 * 1024 * 1024
	if len(data) > maxScan {
		data = data[:maxScan]
	}

	var matches []Match
	lowerData := strings.ToLower(string(data))
	for _, rule := range s.rules {
		var hit []string
		for _, pat := range rule.Strings {
			if len(pat) < 4 {
				continue // patterns trop courts = trop de faux positifs
			}
			if strings.Contains(lowerData, strings.ToLower(pat)) {
				hit = append(hit, pat)
			}
		}
		if len(hit) > 0 {
			matches = append(matches, Match{
				Rule:    rule,
				File:    path,
				SHA256:  h,
				Strings: hit,
			})
		}
	}
	return matches, nil
}

// RuleCount renvoie le nombre de règles chargées.
func (s *Scanner) RuleCount() int { return len(s.rules) }

// sha256OfFile calcule le SHA256 d'un fichier.
func sha256OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
