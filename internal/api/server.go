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

	"github.com/dlnraja/faillefox/internal/core"
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
}

// New crée un serveur lié à 127.0.0.1:port.
func New(engine *core.Engine, driver core.Driver, port int) *Server {
	s := &Server{engine: engine, driver: driver}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/default", s.handleDefault)
	mux.HandleFunc("/api/apps", s.handleApps)
	mux.HandleFunc("/api/events", s.handleEvents) // SSE
	mux.HandleFunc("/api/decide", s.handleDecide) // réponse manuelle à un "ask"

	// UI web embarquée dans le binaire.
	webRoot, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	s.httpSrv = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           mux,
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
