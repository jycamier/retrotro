package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/middleware"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/services"
)

// TeamHandler handles team endpoints
type TeamHandler struct {
	teamService *services.TeamService
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(teamService *services.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}

// List lists all teams for the current user
func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teams, err := h.teamService.ListByUser(ctx, userID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// CreateTeamRequest represents a create team request
type CreateTeamRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

// Create creates a new team
func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" {
		http.Error(w, `{"error": "name and slug are required"}`, http.StatusBadRequest)
		return
	}

	team, err := h.teamService.Create(ctx, userID, services.CreateTeamInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

// Get gets a team by ID
func (h *TeamHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	// Check membership
	isMember, err := h.teamService.IsMember(ctx, teamID, userID)
	if err != nil || !isMember {
		http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
		return
	}

	team, err := h.teamService.GetByID(ctx, teamID)
	if err != nil {
		if err == services.ErrTeamNotFound {
			http.Error(w, `{"error": "team not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// UpdateTeamRequest represents an update team request
type UpdateTeamRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// Update updates a team
func (h *TeamHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	team, err := h.teamService.Update(ctx, userID, teamID, services.UpdateTeamInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if err == services.ErrNotAuthorized {
			http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// Delete deletes a team
func (h *TeamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.teamService.Delete(ctx, userID, teamID); err != nil {
		if err == services.ErrNotAuthorized {
			http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers lists team members
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	members, err := h.teamService.ListMembers(ctx, userID, teamID)
	if err != nil {
		if err == services.ErrNotTeamMember {
			http.Error(w, `{"error": "not a team member"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// AddMemberRequest represents an add member request
type AddMemberRequest struct {
	UserID uuid.UUID   `json:"userId"`
	Role   models.Role `json:"role"`
}

// AddMember adds a member to a team
func (h *TeamHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = models.RoleMember
	}

	if err := h.teamService.AddMember(ctx, userID, teamID, req.UserID, req.Role); err != nil {
		if err == services.ErrNotAuthorized {
			http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveMember removes a member from a team
func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	memberUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.teamService.RemoveMember(ctx, userID, teamID, memberUserID); err != nil {
		if err == services.ErrNotAuthorized {
			http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
			return
		}
		if err == services.ErrCannotLeaveTeam {
			http.Error(w, `{"error": "cannot remove last admin"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateMemberRoleRequest represents an update role request
type UpdateMemberRoleRequest struct {
	Role models.Role `json:"role"`
}

// UpdateMemberRole updates a member's role
func (h *TeamHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	memberUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.teamService.UpdateMemberRole(ctx, userID, teamID, memberUserID, req.Role); err != nil {
		if err == services.ErrNotAuthorized {
			http.Error(w, `{"error": "not authorized"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
