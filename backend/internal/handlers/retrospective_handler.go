package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/middleware"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/services"
)

// RetrospectiveHandler handles retrospective endpoints
type RetrospectiveHandler struct {
	retroService *services.RetrospectiveService
	timerService *services.TimerService
}

// NewRetrospectiveHandler creates a new retrospective handler
func NewRetrospectiveHandler(retroService *services.RetrospectiveService, timerService *services.TimerService) *RetrospectiveHandler {
	return &RetrospectiveHandler{
		retroService: retroService,
		timerService: timerService,
	}
}

// CreateRetroRequest represents a create retrospective request
type CreateRetroRequest struct {
	Name                string                    `json:"name"`
	TeamID              uuid.UUID                 `json:"teamId"`
	TemplateID          uuid.UUID                 `json:"templateId"`
	MaxVotesPerUser     int                       `json:"maxVotesPerUser"`
	MaxVotesPerItem     int                       `json:"maxVotesPerItem"`
	AnonymousVoting     bool                      `json:"anonymousVoting"`
	AnonymousItems      bool                      `json:"anonymousItems"`
	AllowItemEdit       *bool                     `json:"allowItemEdit"`
	AllowVoteChange     *bool                     `json:"allowVoteChange"`
	PhaseTimerOverrides map[models.RetroPhase]int `json:"phaseTimerOverrides"`
	ScheduledAt         *time.Time                `json:"scheduledAt"`
}

// Create creates a new retrospective
func (h *RetrospectiveHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req CreateRetroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.TeamID == uuid.Nil || req.TemplateID == uuid.Nil {
		http.Error(w, `{"error": "name, teamId, and templateId are required"}`, http.StatusBadRequest)
		return
	}

	retro, err := h.retroService.Create(ctx, userID, services.CreateRetroInput{
		Name:                req.Name,
		TeamID:              req.TeamID,
		TemplateID:          req.TemplateID,
		MaxVotesPerUser:     req.MaxVotesPerUser,
		MaxVotesPerItem:     req.MaxVotesPerItem,
		AnonymousVoting:     req.AnonymousVoting,
		AnonymousItems:      req.AnonymousItems,
		AllowItemEdit:       req.AllowItemEdit,
		AllowVoteChange:     req.AllowVoteChange,
		PhaseTimerOverrides: req.PhaseTimerOverrides,
		ScheduledAt:         req.ScheduledAt,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(retro)
}

// List lists retrospectives for a team
func (h *RetrospectiveHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamIDStr := r.URL.Query().Get("teamId")
	if teamIDStr == "" {
		http.Error(w, `{"error": "teamId is required"}`, http.StatusBadRequest)
		return
	}

	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid teamId"}`, http.StatusBadRequest)
		return
	}

	var status *models.RetroStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := models.RetroStatus(statusStr)
		status = &s
	}

	retros, err := h.retroService.ListByTeam(ctx, teamID, status)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retros)
}

// Get gets a retrospective by ID
func (h *RetrospectiveHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		if err == services.ErrRetroNotFound {
			http.Error(w, `{"error": "retrospective not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retro)
}

// Update updates a retrospective
func (h *RetrospectiveHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "retrospective not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		Name                *string                   `json:"name"`
		MaxVotesPerUser     *int                      `json:"maxVotesPerUser"`
		MaxVotesPerItem     *int                      `json:"maxVotesPerItem"`
		AnonymousVoting     *bool                     `json:"anonymousVoting"`
		AnonymousItems      *bool                     `json:"anonymousItems"`
		AllowItemEdit       *bool                     `json:"allowItemEdit"`
		AllowVoteChange     *bool                     `json:"allowVoteChange"`
		PhaseTimerOverrides map[models.RetroPhase]int `json:"phaseTimerOverrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		retro.Name = *req.Name
	}
	if req.MaxVotesPerUser != nil {
		retro.MaxVotesPerUser = *req.MaxVotesPerUser
	}
	if req.MaxVotesPerItem != nil {
		retro.MaxVotesPerItem = *req.MaxVotesPerItem
	}
	if req.AnonymousVoting != nil {
		retro.AnonymousVoting = *req.AnonymousVoting
	}
	if req.AnonymousItems != nil {
		retro.AnonymousItems = *req.AnonymousItems
	}
	if req.AllowItemEdit != nil {
		retro.AllowItemEdit = *req.AllowItemEdit
	}
	if req.AllowVoteChange != nil {
		retro.AllowVoteChange = *req.AllowVoteChange
	}
	if req.PhaseTimerOverrides != nil {
		retro.PhaseTimerOverrides = req.PhaseTimerOverrides
	}

	if err := h.retroService.Update(ctx, retro); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retro)
}

// Delete deletes a retrospective
func (h *RetrospectiveHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.Delete(ctx, retroID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Start starts a retrospective
func (h *RetrospectiveHandler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	retro, err := h.retroService.Start(ctx, retroID)
	if err != nil {
		if errors.Is(err, services.ErrRetroAlreadyStarted) {
			http.Error(w, `{"error": "retrospective already started"}`, http.StatusBadRequest)
			return
		}
		if errors.Is(err, services.ErrRetroNotFound) {
			http.Error(w, `{"error": "retrospective not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retro)
}

// End ends a retrospective
func (h *RetrospectiveHandler) End(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	retro, err := h.retroService.End(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retro)
}

// ListItems lists items for a retrospective
func (h *RetrospectiveHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	items, err := h.retroService.ListItems(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// CreateItemRequest represents a create item request
type CreateItemRequest struct {
	ColumnID string `json:"columnId"`
	Content  string `json:"content"`
}

// CreateItem creates a new item
func (h *RetrospectiveHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	item, err := h.retroService.CreateItem(ctx, retroID, userID, services.CreateItemInput{
		ColumnID: req.ColumnID,
		Content:  req.Content,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

// UpdateItemRequest represents an update item request
type UpdateItemRequest struct {
	Content string `json:"content"`
}

// UpdateItem updates an item
func (h *RetrospectiveHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		http.Error(w, `{"error": "invalid item ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	item, err := h.retroService.UpdateItem(ctx, itemID, req.Content)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

// DeleteItem deletes an item
func (h *RetrospectiveHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		http.Error(w, `{"error": "invalid item ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.DeleteItem(ctx, itemID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GroupItemsRequest represents a group items request
type GroupItemsRequest struct {
	ChildIDs []uuid.UUID `json:"childIds"`
}

// GroupItems groups items together
func (h *RetrospectiveHandler) GroupItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		http.Error(w, `{"error": "invalid item ID"}`, http.StatusBadRequest)
		return
	}

	var req GroupItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.GroupItems(ctx, itemID, req.ChildIDs); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Vote adds a vote to an item
func (h *RetrospectiveHandler) Vote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		http.Error(w, `{"error": "invalid item ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.Vote(ctx, retroID, itemID, userID); err != nil {
		if err == services.ErrVoteLimitReached {
			http.Error(w, `{"error": "vote limit reached"}`, http.StatusBadRequest)
			return
		}
		if err == services.ErrItemVoteLimitReached {
			http.Error(w, `{"error": "item vote limit reached"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Unvote removes a vote from an item
func (h *RetrospectiveHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		http.Error(w, `{"error": "invalid item ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.Unvote(ctx, itemID, userID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListActions lists action items for a retrospective
func (h *RetrospectiveHandler) ListActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	actions, err := h.retroService.ListActions(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actions)
}

// ListTeamActions lists all action items for a team's completed retrospectives
func (h *RetrospectiveHandler) ListTeamActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	actions, err := h.retroService.ListActionsByTeam(ctx, teamID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actions)
}

// CreateActionRequest represents a create action request
type CreateActionRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
	DueDate     *time.Time `json:"dueDate"`
	ItemID      *uuid.UUID `json:"itemId"`
	Priority    int        `json:"priority"`
}

// CreateAction creates a new action item
func (h *RetrospectiveHandler) CreateAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	var req CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	action, err := h.retroService.CreateAction(ctx, retroID, userID, services.CreateActionInput{
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		ItemID:      req.ItemID,
		Priority:    req.Priority,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(action)
}

// UpdateAction updates an action item
func (h *RetrospectiveHandler) UpdateAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		http.Error(w, `{"error": "invalid action ID"}`, http.StatusBadRequest)
		return
	}

	var req CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	action, err := h.retroService.UpdateAction(ctx, actionID, services.CreateActionInput{
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		ItemID:      req.ItemID,
		Priority:    req.Priority,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(action)
}

// DeleteAction deletes an action item
func (h *RetrospectiveHandler) DeleteAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		http.Error(w, `{"error": "invalid action ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.DeleteAction(ctx, actionID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Timer endpoints

// StartTimerRequest represents a start timer request
type StartTimerRequest struct {
	DurationSeconds int `json:"duration_seconds"`
}

// StartTimer starts the timer
func (h *RetrospectiveHandler) StartTimer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	var req StartTimerRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional

	if err := h.timerService.StartTimer(ctx, retroID, req.DurationSeconds); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PauseTimer pauses the timer
func (h *RetrospectiveHandler) PauseTimer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.timerService.PauseTimer(ctx, retroID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ResumeTimer resumes the timer
func (h *RetrospectiveHandler) ResumeTimer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.timerService.ResumeTimer(ctx, retroID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ResetTimer resets the timer
func (h *RetrospectiveHandler) ResetTimer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.timerService.ResetTimer(ctx, retroID); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// AddTimeRequest represents an add time request
type AddTimeRequest struct {
	Seconds int `json:"seconds"`
}

// AddTime adds time to the timer
func (h *RetrospectiveHandler) AddTime(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	var req AddTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.timerService.AddTime(ctx, retroID, req.Seconds); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// NextPhase advances to the next phase
func (h *RetrospectiveHandler) NextPhase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	nextPhase, err := h.retroService.NextPhase(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"phase": string(nextPhase)})
}

// SetPhaseRequest represents a set phase request
type SetPhaseRequest struct {
	Phase string `json:"phase"`
}

// SetPhase sets the current phase
func (h *RetrospectiveHandler) SetPhase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	var req SetPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.retroService.SetPhase(ctx, retroID, models.RetroPhase(req.Phase)); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListTemplates lists templates
func (h *RetrospectiveHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var teamID *uuid.UUID
	if teamIDStr := r.URL.Query().Get("teamId"); teamIDStr != "" {
		id, err := uuid.Parse(teamIDStr)
		if err == nil {
			teamID = &id
		}
	}

	templates, err := h.retroService.ListTemplates(ctx, teamID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(templates)
}

// GetTemplate gets a template by ID
func (h *RetrospectiveHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
	if err != nil {
		http.Error(w, `{"error": "invalid template ID"}`, http.StatusBadRequest)
		return
	}

	template, err := h.retroService.GetTemplate(ctx, templateID)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			http.Error(w, `{"error": "template not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(template)
}

// CreateTemplate creates a new template
func (h *RetrospectiveHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var template models.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	template.ID = uuid.New()
	template.CreatedBy = &userID

	created, err := h.retroService.CreateTemplate(ctx, &template)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// GetRotiResults returns ROTI results for a retrospective
func (h *RetrospectiveHandler) GetRotiResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	results, err := h.retroService.GetRotiResults(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// PatchTeamAction partially updates a team action item (status, assignee)
func (h *RetrospectiveHandler) PatchTeamAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		http.Error(w, `{"error": "invalid action ID"}`, http.StatusBadRequest)
		return
	}

	var req services.PatchActionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	action, err := h.retroService.PatchAction(ctx, actionID, req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(action)
}

// GetIcebreakerMoods returns icebreaker moods for a retrospective
func (h *RetrospectiveHandler) GetIcebreakerMoods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	retroID, err := uuid.Parse(chi.URLParam(r, "retroId"))
	if err != nil {
		http.Error(w, `{"error": "invalid retrospective ID"}`, http.StatusBadRequest)
		return
	}

	moods, err := h.retroService.GetIcebreakerMoods(ctx, retroID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(moods)
}
