package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/claude/freereps/internal/hevy"
)

// handleHevyStatus reports whether a Hevy API key is stored and when the last
// sync ran. The key itself is never returned.
func (s *Server) handleHevyStatus(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUserID(w, r)
	if !ok {
		return
	}

	creds, err := s.db.GetHevyCredentials(r.Context(), uid)
	if err != nil {
		s.log.Error("getting hevy credentials", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read credentials"})
		return
	}

	resp := map[string]any{"configured": creds != nil}
	if creds != nil {
		resp["sync_from"] = creds.SyncFrom.Format("2006-01-02")

		state, err := s.db.GetHevySyncState(r.Context(), uid)
		if err != nil {
			s.log.Warn("getting hevy sync state", "error", err)
		} else if state != nil {
			resp["last_sync"] = state.LastEventAt.Format(time.RFC3339)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleHevyCredentials stores an API key after verifying it against the Hevy
// API. Storing an unverified key would leave the failure to surface hours later
// in a sync log instead of in the form the user is looking at.
func (s *Server) handleHevyCredentials(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		APIKey   string `json:"api_key"`
		SyncFrom string `json:"sync_from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if body.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key is required"})
		return
	}

	syncFrom := time.Now().UTC().Truncate(24 * time.Hour)
	if body.SyncFrom != "" {
		parsed, err := time.Parse("2006-01-02", body.SyncFrom)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sync_from must be YYYY-MM-DD"})
			return
		}
		syncFrom = parsed
	}

	client := hevy.NewClient()
	if _, err := client.GetUserInfo(r.Context(), body.APIKey); err != nil {
		s.log.Warn("hevy api key rejected", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Hevy rejected this API key. It requires an active Hevy Pro subscription.",
		})
		return
	}

	if err := s.db.UpsertHevyCredentials(r.Context(), uid, body.APIKey, syncFrom); err != nil {
		s.log.Error("saving hevy credentials", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// handleHevySync starts a sync in the background and returns immediately.
func (s *Server) handleHevySync(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUserID(w, r)
	if !ok {
		return
	}
	if s.hevySyncer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "hevy sync not configured"})
		return
	}

	// Detached context: the sync outlives the request it was started from.
	go func() {
		if err := s.hevySyncer.TriggerSync(context.Background(), uid); err != nil {
			s.log.Error("hevy sync failed", "user_id", uid, "error", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync_started"})
}

// handleHevyDisconnect removes the stored key and sync state. Sets already
// ingested stay in workout_sets.
func (s *Server) handleHevyDisconnect(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUserID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteHevyCredentials(r.Context(), uid); err != nil {
		s.log.Error("deleting hevy credentials", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to disconnect"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}
