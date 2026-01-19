package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/middleware"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/services"
)

// StatsHandler handles statistics endpoints
type StatsHandler struct {
	statsService *services.StatsService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(statsService *services.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// parseStatsFilter extracts filter parameters from query string
func parseStatsFilter(r *http.Request) *models.StatsFilter {
	filter := &models.StatsFilter{}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	return filter
}

// GetTeamRotiStats returns ROTI statistics for a team
func (h *StatsHandler) GetTeamRotiStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	filter := parseStatsFilter(r)

	stats, err := h.statsService.GetTeamRotiStats(ctx, userID, teamID, filter)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetTeamMoodStats returns mood statistics for a team
func (h *StatsHandler) GetTeamMoodStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	filter := parseStatsFilter(r)

	stats, err := h.statsService.GetTeamMoodStats(ctx, userID, teamID, filter)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetMyStats returns combined statistics for the current user
func (h *StatsHandler) GetMyStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	filter := parseStatsFilter(r)

	stats, err := h.statsService.GetMyStats(ctx, userID, teamID, filter)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUserRotiStats returns ROTI statistics for a specific user
func (h *StatsHandler) GetUserRotiStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}

	filter := parseStatsFilter(r)

	stats, err := h.statsService.GetUserRotiStats(ctx, userID, teamID, targetUserID, filter)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUserMoodStats returns mood statistics for a specific user
func (h *StatsHandler) GetUserMoodStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}

	filter := parseStatsFilter(r)

	stats, err := h.statsService.GetUserMoodStats(ctx, userID, teamID, targetUserID, filter)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
