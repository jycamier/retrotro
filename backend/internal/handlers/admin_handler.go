package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

// AdminHandler handles admin endpoints
type AdminHandler struct {
	userRepo       *postgres.UserRepository
	teamRepo       *postgres.TeamRepository
	teamMemberRepo *postgres.TeamMemberRepository
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(userRepo *postgres.UserRepository, teamRepo *postgres.TeamRepository, teamMemberRepo *postgres.TeamMemberRepository) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		teamRepo:       teamRepo,
		teamMemberRepo: teamMemberRepo,
	}
}

// ListUsers returns all users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	users, err := h.userRepo.ListAll(ctx)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// ListTeams returns all teams with member count
func (h *AdminHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teams, err := h.teamRepo.ListAll(ctx)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Enrich with member count
	result := make([]map[string]interface{}, 0, len(teams))
	for _, team := range teams {
		count, _ := h.teamMemberRepo.CountMembers(ctx, team.ID)
		result = append(result, map[string]interface{}{
			"id":            team.ID,
			"name":          team.Name,
			"slug":          team.Slug,
			"description":   team.Description,
			"isOidcManaged": team.IsOIDCManaged,
			"createdAt":     team.CreatedAt,
			"updatedAt":     team.UpdatedAt,
			"memberCount":   count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetTeamMembers returns all members of a team
func (h *AdminHandler) GetTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	members, err := h.teamMemberRepo.ListByTeam(ctx, teamID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}
