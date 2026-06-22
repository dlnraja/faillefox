// Package api expose le pare-feu via une API HTTP REST + un flux SSE,
// écoutés UNIQUEMENT sur la boucle locale (127.0.0.1). L'UI web se connecte
// à ce serveur.
//
// Sécurité : aucun port n'est jamais bindé sur une interface externe.
// C'est non négociable pour un pare-feu : son canal de contrôle ne doit pas
// être atteignable depuis le réseau.
package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/dlnraja/faillefox/internal/clamscan"
	"github.com/dlnraja/faillefox/internal/core"
	"github.com/dlnraja/faillefox/internal/cvefeed"
	"github.com/dlnraja/faillefox/internal/middleware"
	"github.com/dlnraja/faillefox/internal/refresher"
	"github.com/dlnraja/faillefox/internal/securitycenter"
	"github.com/dlnraja/faillefox/internal/settings"
	"github.com/dlnraja/faillefox/internal/themes"
	"github.com/dlnraja/faillefox/internal/tools"
	"github.com/dlnraja/faillefox/internal/updater"
)

// webFiles embarque toute l'UI (HTML/CSS/JS) dans le binaire final.
// Le serveur sert ainsi un exécutable autonome, sans fichiers externes.
//
//go:embed web/*
var webFiles embed.FS

// Server est le serveur de contrôle du pare-feu.
type Server struct {
	engine  *core.Engine
	driver  core.Driver
	httpSrv *http.Server

	// Modules optionnels v0.3/v0.4 — exposés à l'UI si non nil.
	updater *updater.Updater // nil = auto-update désactivé
	scanner *clamscan.Scanner // nil = ClamAV désactivé
	feed    *cvefeed.Feed     // nil = veille CVE désactivée

	// Centre de sécurité v0.9 — vue unifiée de toutes les protections.
	secCenter *securitycenter.Center

	// Scheduler de rafraîchissement v0.10 — état des référentiels.
	scheduler *refresher.Scheduler

	// Paramètres utilisateur v0.12 — mode simple/avancé + modules.
	settings *settings.Settings
}

// SetUpdater branche l'updater pour l'endpoint /api/updater.
func (s *Server) SetUpdater(u *updater.Updater) { s.updater = u }

// SetScanner branche le scanner ClamAV pour /api/scan.
func (s *Server) SetScanner(sc *clamscan.Scanner) { s.scanner = sc }

// SetFeed branche le feed CVE pour /api/cve.
func (s *Server) SetFeed(f *cvefeed.Feed) { s.feed = f }

// SetSecurityCenter branche le centre de sécurité pour /api/security-center.
func (s *Server) SetSecurityCenter(c *securitycenter.Center) { s.secCenter = c }

// SetScheduler branche le scheduler de rafraîchissement pour /api/refresh-status.
func (s *Server) SetScheduler(sch *refresher.Scheduler) { s.scheduler = sch }

// SetSettings branche les paramètres utilisateur pour /api/settings.
func (s *Server) SetSettings(st *settings.Settings) { s.settings = st }

// New crée un serveur lié à 127.0.0.1:port.
func New(engine *core.Engine, driver core.Driver, port int) *Server {
	s := &Server{engine: engine, driver: driver}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/default", s.handleDefault)
	mux.HandleFunc("/api/apps", s.handleApps)
	mux.HandleFunc("/api/events", s.handleEvents)   // SSE
	mux.HandleFunc("/api/decide", s.handleDecide)   // réponse manuelle à un "ask"
	mux.HandleFunc("/api/updater", s.handleUpdater) // état de l'auto-update
	mux.HandleFunc("/api/scan", s.handleScan)              // scan ClamAV
	mux.HandleFunc("/api/cve", s.handleCVE)                // alertes CVE
	mux.HandleFunc("/api/security-center", s.handleSecCenter) // vue unifiée protections
	mux.HandleFunc("/api/themes", s.handleThemes)             // thèmes UI disponibles
	mux.HandleFunc("/api/refresh-status", s.handleRefresh)    // état du scheduler
	mux.HandleFunc("/api/settings", s.handleSettings)         // paramètres (GET/POST)
	mux.HandleFunc("/api/tools/ports", s.handlePortScan)      // scanner de ports
	mux.HandleFunc("/api/tools/dns-leak", s.handleDNSLeak)    // test fuite DNS
	mux.HandleFunc("/api/tools/password", s.handlePassword)   // vérificateur mot de passe

	// UI web embarquée dans le binaire.
	webRoot, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	// Application des middlewares de sécurité (défense en profondeur) :
	//   1. LoopbackOnly : refuse toute IP non-loopback (même si bind erroné)
	//   2. SecurityHeaders : CSP stricte + durcissement
	//   3. RateLimiter : anti-abus (120 req/min, généreux pour l'UI)
	rateLimiter := middleware.NewRateLimiter(120, time.Minute)
	var handler http.Handler = mux
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.LoopbackOnly(handler)
	handler = rateLimiter.Middleware(handler)

	s.httpSrv = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// ListenAndServe démarre le serveur. Bloquant.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("écoute impossible sur %s: %w", s.httpSrv.Addr, err)
	}
	log.Printf("[api] écoute sur http://%s", s.httpSrv.Addr)
	return s.httpSrv.Serve(ln)
}

// Shutdown arrête proprement le serveur.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// ---- handlers -------------------------------------------------------------

type statusResp struct {
	Driver       string        `json:"driver"`
	Default      core.Decision `json:"default_decision"`
	RulesCount   int           `json:"rules_count"`
	JournalCount int           `json:"journal_count"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := statusResp{
		Driver:       s.driver.Name(),
		Default:      s.engine.DefaultDecision(),
		RulesCount:   len(s.engine.Rules()),
		JournalCount: len(s.engine.RecentEvents(0)),
	}
	writeJSON(w, resp)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.engine.Rules())
	case http.MethodPost:
		var rule core.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		saved, err := s.engine.AddRule(rule)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, saved)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "paramètre id requis", http.StatusBadRequest)
			return
		}
		if err := s.engine.DeleteRule(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Decision core.Decision `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}
	switch body.Decision {
	case core.DecisionAllow, core.DecisionDeny, core.DecisionAsk:
	default:
		http.Error(w, "décision invalide (allow/deny/ask)", http.StatusBadRequest)
		return
	}
	if err := s.engine.SetDefaultDecision(body.Decision); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"decision": body.Decision})
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.driver.ListApps()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, apps)
}

// handleDecide permet à l'UI de répondre à une connexion en attente ("ask").
// En v1 (stub) c'est surtout utilitaire ; les vrais backends l'utiliseront
// pour lever les blocages en attente de réponse utilisateur.
func (s *Server) handleDecide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		AppID    string        `json:"app_id"`
		Decision core.Decision `json:"decision"`
		Forever  bool          `json:"forever"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}
	if body.Forever {
		if _, err := s.engine.AddRule(core.Rule{
			AppID:  body.AppID,
			Action: body.Decision,
			Note:   "ajouté depuis l'UI",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// handleEvents expose le flux d'événements en Server-Sent Events.
// Le navigateur s'y abonne avec `new EventSource("/api/events")`.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming non supporté", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id, ch := s.engine.Subscribe(64)
	defer s.engine.Unsubscribe(id)

	// Envoi initial : derniers événements connus.
	recent := s.engine.RecentEvents(50)
	for _, ev := range recent {
		writeSSE(w, ev)
	}
	flusher.Flush()

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, ev)
			flusher.Flush()
		case <-keepAlive.C:
			// keep-alive SSE : le retour d'erreur n'est pas récupérable
			// (client déconnecté), on l'ignore volontairement.
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// ---- handlers v0.3/v0.4 ---------------------------------------------------

// handleUpdater expose l'état de l'auto-update (dernier fetch, nb domaines...).
func (s *Server) handleUpdater(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		writeJSON(w, map[string]any{"enabled": false})
		return
	}
	writeJSON(w, map[string]any{
		"enabled": true,
		"status":  s.updater.Status(),
	})
}

// handleScan déclenche un scan ClamAV sur un fichier fourni en query ?path=.
// Renvoie le résultat du scan (infecté ?, signature, détail).
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		http.Error(w, "scanner ClamAV non configuré (lancez avec -clamav)", http.StatusServiceUnavailable)
		return
	}
	if !s.scanner.IsAvailable() {
		http.Error(w, "ClamAV non installé (voir docs/clamav.md)", http.StatusServiceUnavailable)
		return
	}
	path := r.URL.Query().Get("path")
	// Sécurité : sanitize le chemin pour bloquer le path traversal.
	path = middleware.SanitizePath(path)
	if path == "" {
		http.Error(w, "paramètre path invalide (path traversal bloqué)", http.StatusBadRequest)
		return
	}
	result, err := s.scanner.ScanFile(r.Context(), path)
	if err != nil {
		http.Error(w, "scan échoué: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// handleCVE renvoie les alertes CVE pour les logiciels fournis en POST.
// Body JSON: {"software": [{"name":"curl","version":"7.74.0"}, ...]}
// Réponse: liste d'alertes.
func (s *Server) handleCVE(w http.ResponseWriter, r *http.Request) {
	if s.feed == nil {
		writeJSON(w, map[string]any{"enabled": false, "alerts": []any{}})
		return
	}
	if r.Method != http.MethodPost {
		// GET : on renvoie juste l'état.
		writeJSON(w, map[string]any{"enabled": true})
		return
	}
	var body struct {
		Software []cvefeed.Software `json:"software"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	alerts := s.feed.CheckSoftware(body.Software)
	writeJSON(w, map[string]any{"alerts": alerts})
}

// handleSecCenter expose la vue unifiée de toutes les protections :
// résumé (score, nb actives/inactives) + état détaillé de chaque couche.
func (s *Server) handleSecCenter(w http.ResponseWriter, r *http.Request) {
	if s.secCenter == nil {
		// Si pas de centre branché, on renvoie un état vide mais valide.
		writeJSON(w, map[string]any{
			"summary":     map[string]any{"total": 0, "active": 0, "score": 0},
			"protections": []any{},
		})
		return
	}
	writeJSON(w, map[string]any{
		"summary":     s.secCenter.GetSummary(),
		"protections": s.secCenter.States(),
	})
}

// handleThemes expose la liste des thèmes UI disponibles.
func (s *Server) handleThemes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"available": themes.List(),
		"default":   themes.Default,
	})
}

// handleRefresh expose l'état du scheduler de rafraîchissement (quand chaque
// référentiel a été rafraîchi pour la dernière fois, prochain échéance, etc.).
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		writeJSON(w, map[string]any{"enabled": false, "sources": []any{}})
		return
	}
	writeJSON(w, map[string]any{
		"enabled": true,
		"sources": s.scheduler.Status(),
	})
}

// handleSettings expose (GET) et met à jour (POST) les paramètres utilisateur.
// LoopbackOnly est toujours forcé à true (non négociable) même si le client
// tente de le désactiver.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if s.settings == nil {
		http.Error(w, "settings non configurés", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.settings)
	case http.MethodPost:
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		// LoopbackOnly est non négociable : on le retire du patch client.
		delete(patch, "loopback_only")
		if err := s.settings.Update(patch); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, s.settings)
	default:
		http.Error(w, "GET ou POST attendu", http.StatusMethodNotAllowed)
	}
}

// ---- handlers outils gratuits (v0.14) ------------------------------------

// handlePortScan scanne les ports ouverts sur localhost (surface d'attaque).
// Sécurité : on force host=127.0.0.1 pour éviter tout scan externe via l'API.
func (s *Server) handlePortScan(w http.ResponseWriter, r *http.Request) {
	scanner := tools.NewPortScanner()
	// Sécurité : UNIQUEMENT localhost (pas de scan d'hôtes externes via API).
	results := scanner.Scan("127.0.0.1", 2*time.Second)
	writeJSON(w, map[string]any{
		"host":      "127.0.0.1",
		"ports":     results,
		"open_count": len(results),
	})
}

// handleDNSLeak teste quels résolveurs DNS répondent (détection de fuite).
func (s *Server) handleDNSLeak(w http.ResponseWriter, r *http.Request) {
	tester := tools.NewDNSLeakTest()
	resolvers := tester.Test()
	writeJSON(w, map[string]any{
		"resolvers": resolvers,
	})
}

// handlePassword évalue la force d'un mot de passe fourni en POST.
// SéCURITÉ : le mot de passe n'est JAMAIS stocké ni loggé — on ne renvoie
// que des métriques (score, entropie, feedback).
func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}
	checker := tools.NewPasswordChecker()
	strength := checker.Evaluate(body.Password)
	// On ne renvoie JAMAIS le mot de passe dans la réponse.
	writeJSON(w, strength)
}

// ---- helpers --------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeSSE(w http.ResponseWriter, ev core.Event) {
	data, _ := json.Marshal(ev)
	// Le retour d'erreur d'écriture n'est pas récupérable ici.
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}

// PortFromArgs parse un numéro de port depuis une chaîne.
func PortFromArgs(s string, def int) int {
	if s == "" {
		return def
	}
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 {
		return def
	}
	return p
}
