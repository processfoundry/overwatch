package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/christianmscott/overwatch/internal/auth"
	"github.com/christianmscott/overwatch/internal/checks"
	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/internal/results"
	"github.com/christianmscott/overwatch/pkg/spec"
)

type Server struct {
	mu         sync.RWMutex
	cfg        *spec.Config
	cfgPath    string
	results    *results.Store
	httpServer *http.Server
	onReload   func()
}

func New(cfg *spec.Config, cfgPath string, store *results.Store) *Server {
	s := &Server{
		cfg:     cfg,
		cfgPath: cfgPath,
		results: store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/join", s.handleJoin)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/checks", s.handleListChecks)
	mux.HandleFunc("POST /api/checks", s.handleAddCheck)
	mux.HandleFunc("PUT /api/checks/{name}", s.handleUpdateCheck)
	mux.HandleFunc("DELETE /api/checks/{name}", s.handleRemoveCheck)
	mux.HandleFunc("GET /api/alerts", s.handleListAlerts)
	mux.HandleFunc("POST /api/alerts", s.handleAddAlert)
	mux.HandleFunc("PUT /api/alerts/{name}", s.handleUpdateAlert)
	mux.HandleFunc("DELETE /api/alerts/{name}", s.handleRemoveAlert)
	mux.HandleFunc("POST /api/checkin/{name}", s.handleCheckIn)
	mux.HandleFunc("POST /api/reload", s.handleReload)

	addr := net.JoinHostPort(cfg.Server.BindAddress, fmt.Sprintf("%d", cfg.Server.BindPort))
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.authMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Addr() string { return s.httpServer.Addr }

func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutCtx)
	}()

	slog.Info("api listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) UpdateConfig(cfg *spec.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *Server) OnReload(fn func()) {
	s.onReload = fn
}

func (s *Server) handleReload(w http.ResponseWriter, _ *http.Request) {
	if s.onReload != nil {
		s.onReload()
		writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
	} else {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reload not configured"})
	}
}

var publicPaths = map[string]bool{
	"/api/health": true,
	"/api/join":   true,
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, "/api/checkin/") {
			next.ServeHTTP(w, r)
			return
		}
		s.mu.RLock()
		keys := s.cfg.Server.AuthorizedUsers
		joinToken := s.cfg.Server.JoinToken
		s.mu.RUnlock()

		if len(keys) == 0 && joinToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		if len(keys) == 0 {
			slog.Warn("auth denied: no authorized users configured", "path", r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized: no authorized users configured"})
			return
		}
		if err := auth.VerifyRequest(r, keys); err != nil {
			slog.Warn("auth failed", "error", err, "path", r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized: " + err.Error()})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JoinToken string `json:"join_token"`
		PublicKey string `json:"public_key"`
		Label     string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if req.JoinToken != s.cfg.Server.JoinToken {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid join token"})
		return
	}

	pubBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid public key encoding"})
		return
	}

	keyID := auth.KeyID(pubBytes)

	for _, k := range s.cfg.Server.AuthorizedUsers {
		if k.KeyID == keyID {
			writeJSON(w, http.StatusOK, map[string]string{"key_id": keyID, "message": "already joined"})
			return
		}
	}

	s.cfg.Server.AuthorizedUsers = append(s.cfg.Server.AuthorizedUsers, spec.PublicKeyEntry{
		KeyID:     keyID,
		PublicKey: req.PublicKey,
		Label:     req.Label,
	})
	if err := config.Save(s.cfgPath, s.cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("client joined", "key_id", keyID, "label", req.Label)
	writeJSON(w, http.StatusOK, map[string]string{"key_id": keyID, "message": "joined"})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	latest := s.results.All()
	type checkStatus struct {
		spec.CheckSpec
		LastResult *spec.CheckResult `json:"last_result,omitempty"`
	}
	var out []checkStatus
	for _, c := range cfg.Checks {
		cs := checkStatus{CheckSpec: c}
		if r, ok := latest[c.Name]; ok {
			cs.LastResult = &r
		}
		out = append(out, cs)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checks": out,
		"alerts": cfg.Alerts,
		"server": cfg.Server,
	})
}

// --- Checks CRUD ---

func (s *Server) handleListChecks(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, s.cfg.Checks)
}

func (s *Server) handleAddCheck(w http.ResponseWriter, r *http.Request) {
	var c spec.CheckSpec
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.cfg.Checks {
		if existing.Name == c.Name {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "check already exists: " + c.Name})
			return
		}
	}
	s.cfg.Checks = append(s.cfg.Checks, c)
	if err := config.Save(s.cfgPath, s.cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var patch spec.CheckSpec
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cfg.Checks {
		if c.Name == name {
			if patch.Type != "" {
				s.cfg.Checks[i].Type = patch.Type
			}
			if patch.Target != "" {
				s.cfg.Checks[i].Target = patch.Target
			}
			if patch.Interval.Duration != 0 {
				s.cfg.Checks[i].Interval = patch.Interval
			}
			if patch.Timeout.Duration != 0 {
				s.cfg.Checks[i].Timeout = patch.Timeout
			}
			if patch.ExpectedStatus != 0 {
				s.cfg.Checks[i].ExpectedStatus = patch.ExpectedStatus
			}
			if patch.MaxSilence.Duration != 0 {
				s.cfg.Checks[i].MaxSilence = patch.MaxSilence
			}
			if patch.Alerts != nil {
				s.cfg.Checks[i].Alerts = patch.Alerts
			}
			if patch.Headers != nil {
				s.cfg.Checks[i].Headers = patch.Headers
			}
			if err := config.Save(s.cfgPath, s.cfg); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, s.cfg.Checks[i])
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "check not found: " + name})
}

func (s *Server) handleRemoveCheck(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cfg.Checks {
		if c.Name == name {
			s.cfg.Checks = append(s.cfg.Checks[:i], s.cfg.Checks[i+1:]...)
			if err := config.Save(s.cfgPath, s.cfg); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"removed": name})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "check not found: " + name})
}

// --- Alerts CRUD ---

func (s *Server) handleListAlerts(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, s.cfg.Alerts)
}

func (s *Server) handleAddAlert(w http.ResponseWriter, r *http.Request) {
	var wh spec.WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&wh); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.cfg.Alerts.Webhooks {
		if existing.Name == wh.Name {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "alert already exists: " + wh.Name})
			return
		}
	}
	s.cfg.Alerts.Webhooks = append(s.cfg.Alerts.Webhooks, wh)
	if err := config.Save(s.cfgPath, s.cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, wh)
}

func (s *Server) handleUpdateAlert(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var patch spec.WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, wh := range s.cfg.Alerts.Webhooks {
		if wh.Name == name {
			if patch.URL != "" {
				s.cfg.Alerts.Webhooks[i].URL = patch.URL
			}
			if patch.Method != "" {
				s.cfg.Alerts.Webhooks[i].Method = patch.Method
			}
			if patch.Timeout.Duration != 0 {
				s.cfg.Alerts.Webhooks[i].Timeout = patch.Timeout
			}
			if patch.Headers != nil {
				s.cfg.Alerts.Webhooks[i].Headers = patch.Headers
			}
			if err := config.Save(s.cfgPath, s.cfg); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, s.cfg.Alerts.Webhooks[i])
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found: " + name})
}

func (s *Server) handleRemoveAlert(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, wh := range s.cfg.Alerts.Webhooks {
		if wh.Name == name {
			s.cfg.Alerts.Webhooks = append(s.cfg.Alerts.Webhooks[:i], s.cfg.Alerts.Webhooks[i+1:]...)
			if err := config.Save(s.cfgPath, s.cfg); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"removed": name})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found: " + name})
}

// --- Check-in webhook ---

func (s *Server) handleCheckIn(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.RLock()
	found := false
	for _, c := range s.cfg.Checks {
		if c.Name == name && c.Type == spec.CheckCheckIn {
			found = true
			break
		}
	}
	s.mu.RUnlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "checkin check not found: " + name})
		return
	}

	fail := strings.EqualFold(r.URL.Query().Get("status"), "fail")
	if fail {
		checks.DefaultCheckIn.RecordFailure(name)
		writeJSON(w, http.StatusOK, map[string]string{"recorded": "failure", "check": name})
	} else {
		checks.DefaultCheckIn.RecordPing(name)
		writeJSON(w, http.StatusOK, map[string]string{"recorded": "ok", "check": name})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
