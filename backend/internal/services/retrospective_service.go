package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var (
	ErrRetroNotFound        = errors.New("retrospective not found")
	ErrItemNotFound         = errors.New("item not found")
	ErrActionNotFound       = errors.New("action item not found")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrVoteLimitReached     = errors.New("vote limit reached")
	ErrItemVoteLimitReached = errors.New("item vote limit reached")
	ErrInvalidPhase         = errors.New("invalid phase for this operation")
)

// RetrospectiveService handles retrospective operations
type RetrospectiveService struct {
	retroRepo      *postgres.RetrospectiveRepository
	templateRepo   *postgres.TemplateRepository
	itemRepo       *postgres.ItemRepository
	voteRepo       *postgres.VoteRepository
	actionRepo     *postgres.ActionItemRepository
	icebreakerRepo *postgres.IcebreakerRepository
	rotiRepo       *postgres.RotiRepository
	webhookService *WebhookService
}

// NewRetrospectiveService creates a new retrospective service
func NewRetrospectiveService(
	retroRepo *postgres.RetrospectiveRepository,
	templateRepo *postgres.TemplateRepository,
	itemRepo *postgres.ItemRepository,
	voteRepo *postgres.VoteRepository,
	actionRepo *postgres.ActionItemRepository,
	icebreakerRepo *postgres.IcebreakerRepository,
	rotiRepo *postgres.RotiRepository,
	webhookService *WebhookService,
) *RetrospectiveService {
	return &RetrospectiveService{
		retroRepo:      retroRepo,
		templateRepo:   templateRepo,
		itemRepo:       itemRepo,
		voteRepo:       voteRepo,
		actionRepo:     actionRepo,
		icebreakerRepo: icebreakerRepo,
		rotiRepo:       rotiRepo,
		webhookService: webhookService,
	}
}

// CreateRetroInput represents input for creating a retrospective
type CreateRetroInput struct {
	Name                  string
	TeamID                uuid.UUID
	TemplateID            uuid.UUID
	SessionType           models.SessionType
	MaxVotesPerUser       int
	MaxVotesPerItem       int
	AnonymousVoting       bool
	AnonymousItems        bool
	AllowItemEdit         *bool // Pointer to distinguish between false and not-set (defaults to true)
	AllowVoteChange       *bool // Pointer to distinguish between false and not-set (defaults to true)
	PhaseTimerOverrides   map[models.RetroPhase]int
	ScheduledAt           *time.Time
	LCTopicTimeboxSeconds *int
}

// Create creates a new retrospective
func (s *RetrospectiveService) Create(ctx context.Context, facilitatorID uuid.UUID, input CreateRetroInput) (*models.Retrospective, error) {
	// Verify template exists
	_, err := s.templateRepo.FindByID(ctx, input.TemplateID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	maxVotes := input.MaxVotesPerUser
	if maxVotes <= 0 {
		maxVotes = 5
	}

	maxVotesPerItem := input.MaxVotesPerItem
	if maxVotesPerItem <= 0 {
		maxVotesPerItem = 3
	}

	// Default to true if not explicitly set
	allowItemEdit := true
	if input.AllowItemEdit != nil {
		allowItemEdit = *input.AllowItemEdit
	}

	allowVoteChange := true
	if input.AllowVoteChange != nil {
		allowVoteChange = *input.AllowVoteChange
	}

	// Default session type to retro
	sessionType := input.SessionType
	if sessionType == "" {
		sessionType = models.SessionTypeRetro
	}

	retro := &models.Retrospective{
		ID:                    uuid.New(),
		Name:                  input.Name,
		TeamID:                input.TeamID,
		TemplateID:            input.TemplateID,
		FacilitatorID:         facilitatorID,
		Status:                models.StatusDraft,
		CurrentPhase:          models.PhaseBrainstorm,
		MaxVotesPerUser:       maxVotes,
		MaxVotesPerItem:       maxVotesPerItem,
		AnonymousVoting:       input.AnonymousVoting,
		AnonymousItems:        input.AnonymousItems,
		AllowItemEdit:         allowItemEdit,
		AllowVoteChange:       allowVoteChange,
		PhaseTimerOverrides:   input.PhaseTimerOverrides,
		ScheduledAt:           input.ScheduledAt,
		SessionType:           sessionType,
		LCTopicTimeboxSeconds: input.LCTopicTimeboxSeconds,
	}

	return s.retroRepo.Create(ctx, retro)
}

// GetByID gets a retrospective by ID
func (s *RetrospectiveService) GetByID(ctx context.Context, id uuid.UUID) (*models.Retrospective, error) {
	retro, err := s.retroRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrRetroNotFound
		}
		return nil, err
	}
	return retro, nil
}

// ListByTeam lists retrospectives for a team
func (s *RetrospectiveService) ListByTeam(ctx context.Context, teamID uuid.UUID, status *models.RetroStatus) ([]*models.Retrospective, error) {
	return s.retroRepo.ListByTeam(ctx, teamID, status)
}

var ErrRetroAlreadyStarted = errors.New("retrospective already started")

// Start starts a retrospective
func (s *RetrospectiveService) Start(ctx context.Context, id uuid.UUID) (*models.Retrospective, error) {
	retro, err := s.retroRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("Start: failed to find retro %s: %v", id, err)
		return nil, err
	}

	log.Printf("Start: retro %s current status=%s, phase=%s", id, retro.Status, retro.CurrentPhase)

	// If already active, return the current state (idempotent behavior)
	if retro.Status == models.StatusActive {
		log.Printf("Start: retro %s already active, returning current state", id)
		return retro, nil
	}

	if retro.Status != models.StatusDraft {
		log.Printf("Start: retro %s has invalid status %s for starting", id, retro.Status)
		return nil, ErrRetroAlreadyStarted
	}

	now := time.Now()
	retro.Status = models.StatusActive
	retro.StartedAt = &now
	retro.CurrentPhase = models.PhaseWaiting

	if err := s.retroRepo.Update(ctx, retro); err != nil {
		log.Printf("Start: failed to update retro %s: %v", id, err)
		return nil, err
	}

	log.Printf("Start: retro %s successfully started", id)
	return retro, nil
}

// End ends a retrospective
func (s *RetrospectiveService) End(ctx context.Context, id uuid.UUID) (*models.Retrospective, error) {
	retro, err := s.retroRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	retro.Status = models.StatusCompleted
	retro.EndedAt = &now

	if err := s.retroRepo.Update(ctx, retro); err != nil {
		return nil, err
	}

	// Dispatch retro.completed webhook asynchronously
	if s.webhookService != nil {
		go s.dispatchRetroCompletedWebhook(ctx, retro)
	}

	return retro, nil
}

// dispatchRetroCompletedWebhook gathers data and dispatches the retro.completed webhook
func (s *RetrospectiveService) dispatchRetroCompletedWebhook(ctx context.Context, retro *models.Retrospective) {
	// Gather items
	items, err := s.itemRepo.ListByRetro(ctx, retro.ID)
	if err != nil {
		log.Printf("webhook: failed to list items for retro %s: %v", retro.ID, err)
		items = []*models.Item{}
	}

	// Gather actions
	actions, err := s.actionRepo.ListByRetro(ctx, retro.ID)
	if err != nil {
		log.Printf("webhook: failed to list actions for retro %s: %v", retro.ID, err)
		actions = []*models.ActionItem{}
	}

	// Gather moods
	moods, err := s.icebreakerRepo.ListMoods(ctx, retro.ID)
	if err != nil {
		log.Printf("webhook: failed to list moods for retro %s: %v", retro.ID, err)
		moods = []*models.IcebreakerMood{}
	}

	// Gather ROTI votes
	rotiVotes, err := s.rotiRepo.ListVotes(ctx, retro.ID)
	if err != nil {
		log.Printf("webhook: failed to list roti votes for retro %s: %v", retro.ID, err)
		rotiVotes = []*models.RotiVote{}
	}

	// Calculate average ROTI
	var averageRoti float64
	if len(rotiVotes) > 0 {
		var total int
		for _, v := range rotiVotes {
			total += v.Rating
		}
		averageRoti = float64(total) / float64(len(rotiVotes))
	}

	// Convert moods to webhook format
	webhookMoods := make([]models.MoodData, 0, len(moods))
	for _, m := range moods {
		webhookMoods = append(webhookMoods, models.MoodData{
			UserID: m.UserID,
			Mood:   m.Mood,
		})
	}

	// Convert ROTI votes to webhook format
	webhookRotiVotes := make([]models.RotiVoteData, 0, len(rotiVotes))
	for _, v := range rotiVotes {
		webhookRotiVotes = append(webhookRotiVotes, models.RotiVoteData{
			UserID: v.UserID,
			Rating: v.Rating,
		})
	}

	// Dispatch webhook
	var avgRotiPtr *float64
	if len(rotiVotes) > 0 {
		avgRotiPtr = &averageRoti
	}

	s.webhookService.DispatchRetroCompleted(ctx, retro, models.RetroCompletedData{
		Name:             retro.Name,
		FacilitatorID:    retro.FacilitatorID,
		ParticipantCount: len(moods), // Use mood count as participant proxy
		ItemCount:        len(items),
		ActionCount:      len(actions),
		AverageRoti:      avgRotiPtr,
		Moods:            webhookMoods,
		RotiVotes:        webhookRotiVotes,
	})
}

// Update updates a retrospective
func (s *RetrospectiveService) Update(ctx context.Context, retro *models.Retrospective) error {
	return s.retroRepo.Update(ctx, retro)
}

// Delete deletes a retrospective
func (s *RetrospectiveService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.retroRepo.Delete(ctx, id)
}

// SetPhase sets the current phase
func (s *RetrospectiveService) SetPhase(ctx context.Context, id uuid.UUID, phase models.RetroPhase) error {
	return s.retroRepo.UpdatePhase(ctx, id, phase)
}

// GetPhaseSequence returns the phase sequence for a given session type
func GetPhaseSequence(sessionType models.SessionType) []models.RetroPhase {
	if sessionType == models.SessionTypeLeanCoffee {
		return []models.RetroPhase{
			models.PhaseWaiting,
			models.PhaseIcebreaker,
			models.PhasePropose,
			models.PhaseVote,
			models.PhaseDiscuss,
			models.PhaseRoti,
		}
	}
	// Default retro phases
	return []models.RetroPhase{
		models.PhaseWaiting,
		models.PhaseIcebreaker,
		models.PhaseBrainstorm,
		models.PhaseGroup,
		models.PhaseVote,
		models.PhaseDiscuss,
		models.PhaseRoti,
	}
}

// NextPhase advances to the next phase
func (s *RetrospectiveService) NextPhase(ctx context.Context, id uuid.UUID) (models.RetroPhase, error) {
	retro, err := s.retroRepo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}

	phases := GetPhaseSequence(retro.SessionType)

	currentIdx := -1
	for i, p := range phases {
		if p == retro.CurrentPhase {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 || currentIdx >= len(phases)-1 {
		return retro.CurrentPhase, nil // Already at last phase
	}

	nextPhase := phases[currentIdx+1]
	if err := s.retroRepo.UpdatePhase(ctx, id, nextPhase); err != nil {
		return "", err
	}

	return nextPhase, nil
}

// GetPhaseDuration gets the default duration for a phase
func (s *RetrospectiveService) GetPhaseDuration(ctx context.Context, templateID uuid.UUID, phase models.RetroPhase) (int, error) {
	template, err := s.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return 0, err
	}

	if duration, ok := template.PhaseTimes[phase]; ok {
		return duration, nil
	}

	// Default durations
	defaults := map[models.RetroPhase]int{
		models.PhaseWaiting:    0,
		models.PhaseIcebreaker: 120,
		models.PhaseBrainstorm: 300,
		models.PhaseGroup:      180,
		models.PhaseVote:       180,
		models.PhaseDiscuss:    900,
		models.PhaseRoti:       120,
		models.PhasePropose:    300,
	}

	return defaults[phase], nil
}

// CreateItemInput represents input for creating an item
type CreateItemInput struct {
	ColumnID string
	Content  string
}

// CreateItem creates a new item
func (s *RetrospectiveService) CreateItem(ctx context.Context, retroID, authorID uuid.UUID, input CreateItemInput) (*models.Item, error) {
	position, err := s.itemRepo.GetNextPosition(ctx, retroID, input.ColumnID)
	if err != nil {
		return nil, err
	}

	item := &models.Item{
		ID:       uuid.New(),
		RetroID:  retroID,
		ColumnID: input.ColumnID,
		Content:  input.Content,
		AuthorID: authorID,
		Position: position,
	}

	return s.itemRepo.Create(ctx, item)
}

// UpdateItem updates an item
func (s *RetrospectiveService) UpdateItem(ctx context.Context, id uuid.UUID, content string) (*models.Item, error) {
	item, err := s.itemRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	item.Content = content
	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// DeleteItem deletes an item
func (s *RetrospectiveService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return s.itemRepo.Delete(ctx, id)
}

// MoveItem moves an item to a new position
func (s *RetrospectiveService) MoveItem(ctx context.Context, id uuid.UUID, columnID string, position int) (*models.Item, error) {
	item, err := s.itemRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	item.ColumnID = columnID
	item.Position = position
	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// GroupItems groups items together
func (s *RetrospectiveService) GroupItems(ctx context.Context, parentID uuid.UUID, childIDs []uuid.UUID) ([]uuid.UUID, error) {
	log.Printf("GroupItems: parentID=%s, childIDs=%v", parentID, childIDs)
	allAffected := make([]uuid.UUID, 0, len(childIDs))
	for _, childID := range childIDs {
		item, err := s.itemRepo.FindByID(ctx, childID)
		if err != nil {
			log.Printf("GroupItems: FindByID failed for %s: %v", childID, err)
			continue
		}

		// When re-grouping an item that already has grouped children,
		// move those children to the new parent as well
		allItems, err := s.itemRepo.ListByRetro(ctx, item.RetroID)
		if err != nil {
			log.Printf("GroupItems: Failed to list items for retro %s: %v", item.RetroID, err)
		} else {
			for _, existingItem := range allItems {
				if existingItem.GroupID != nil && *existingItem.GroupID == childID {
					existingItem.GroupID = &parentID
					if err := s.itemRepo.Update(ctx, existingItem); err != nil {
						log.Printf("GroupItems: Failed to move item %s to new group: %v", existingItem.ID, err)
					} else {
						allAffected = append(allAffected, existingItem.ID)
					}
				}
			}
		}

		item.GroupID = &parentID
		if err := s.itemRepo.Update(ctx, item); err != nil {
			log.Printf("GroupItems: Update failed for %s: %v", childID, err)
		} else {
			allAffected = append(allAffected, childID)
		}
	}
	return allAffected, nil
}

// ListItems lists items for a retrospective
func (s *RetrospectiveService) ListItems(ctx context.Context, retroID uuid.UUID) ([]*models.Item, error) {
	return s.itemRepo.ListByRetro(ctx, retroID)
}

// Vote adds a vote to an item
func (s *RetrospectiveService) Vote(ctx context.Context, retroID, itemID, userID uuid.UUID) error {
	retro, err := s.retroRepo.FindByID(ctx, retroID)
	if err != nil {
		return err
	}

	// Check total vote limit per user in the retro
	currentVotes, err := s.voteRepo.CountByUser(ctx, retroID, userID)
	if err != nil {
		return err
	}

	if currentVotes >= retro.MaxVotesPerUser {
		return ErrVoteLimitReached
	}

	// Check vote limit per item
	votesOnItem, err := s.voteRepo.CountByUserOnItem(ctx, itemID, userID)
	if err != nil {
		return err
	}

	if votesOnItem >= retro.MaxVotesPerItem {
		return ErrItemVoteLimitReached
	}

	vote := &models.Vote{
		ID:     uuid.New(),
		ItemID: itemID,
		UserID: userID,
	}

	_, err = s.voteRepo.Create(ctx, vote)
	return err
}

// Unvote removes a vote from an item
func (s *RetrospectiveService) Unvote(ctx context.Context, itemID, userID uuid.UUID) error {
	return s.voteRepo.Delete(ctx, itemID, userID)
}

// HasVoted checks if a user has voted on an item
func (s *RetrospectiveService) HasVoted(ctx context.Context, itemID, userID uuid.UUID) (bool, error) {
	return s.voteRepo.HasVoted(ctx, itemID, userID)
}

// GetUserVoteCount gets the number of votes a user has used
func (s *RetrospectiveService) GetUserVoteCount(ctx context.Context, retroID, userID uuid.UUID) (int, error) {
	return s.voteRepo.CountByUser(ctx, retroID, userID)
}

// GetUserVoteCountOnItem gets the number of votes a user has on a specific item
func (s *RetrospectiveService) GetUserVoteCountOnItem(ctx context.Context, itemID, userID uuid.UUID) (int, error) {
	return s.voteRepo.CountByUserOnItem(ctx, itemID, userID)
}

// GetVoteSummary returns the vote summary for a retrospective: map[userID]map[itemID]count
func (s *RetrospectiveService) GetVoteSummary(ctx context.Context, retroID uuid.UUID) (map[uuid.UUID]map[uuid.UUID]int, error) {
	return s.voteRepo.GetVoteSummaryByRetro(ctx, retroID)
}

// CreateActionInput represents input for creating an action item
type CreateActionInput struct {
	Title       string
	Description *string
	AssigneeID  *uuid.UUID
	DueDate     *time.Time
	ItemID      *uuid.UUID
	Priority    int
}

// PatchActionInput represents input for partially updating an action item
type PatchActionInput struct {
	Status      *string    `json:"status"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
	Description *string    `json:"description"`
}

// CreateAction creates a new action item
func (s *RetrospectiveService) CreateAction(ctx context.Context, retroID, createdBy uuid.UUID, input CreateActionInput) (*models.ActionItem, error) {
	action := &models.ActionItem{
		ID:          uuid.New(),
		RetroID:     retroID,
		ItemID:      input.ItemID,
		Title:       input.Title,
		Description: input.Description,
		AssigneeID:  input.AssigneeID,
		DueDate:     input.DueDate,
		Priority:    input.Priority,
		CreatedBy:   createdBy,
		Status:      "todo",
	}

	createdAction, err := s.actionRepo.Create(ctx, action)
	if err != nil {
		return nil, err
	}

	// Dispatch action.created webhook asynchronously
	if s.webhookService != nil {
		go s.dispatchActionCreatedWebhook(ctx, createdAction, retroID)
	}

	return createdAction, nil
}

// dispatchActionCreatedWebhook dispatches the action.created webhook
func (s *RetrospectiveService) dispatchActionCreatedWebhook(ctx context.Context, action *models.ActionItem, retroID uuid.UUID) {
	// Get the retro to find the team ID
	retro, err := s.retroRepo.FindByID(ctx, retroID)
	if err != nil {
		log.Printf("webhook: failed to find retro %s for action webhook: %v", retroID, err)
		return
	}

	data := models.ActionCreatedData{
		ActionID:     action.ID,
		Title:        action.Title,
		Description:  action.Description,
		AssigneeID:   action.AssigneeID,
		DueDate:      action.DueDate,
		Priority:     action.Priority,
		CreatedBy:    action.CreatedBy,
		SourceItemID: action.ItemID,
	}

	s.webhookService.DispatchActionCreated(ctx, action, retro.TeamID, data)
}

// UpdateAction updates an action item
func (s *RetrospectiveService) UpdateAction(ctx context.Context, id uuid.UUID, input CreateActionInput) (*models.ActionItem, error) {
	action, err := s.actionRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrActionNotFound
		}
		return nil, err
	}

	action.Title = input.Title
	action.Description = input.Description
	action.AssigneeID = input.AssigneeID
	action.DueDate = input.DueDate
	action.Priority = input.Priority

	if err := s.actionRepo.Update(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// CompleteAction marks an action item as completed
func (s *RetrospectiveService) CompleteAction(ctx context.Context, id uuid.UUID) (*models.ActionItem, error) {
	action, err := s.actionRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrActionNotFound
		}
		return nil, err
	}

	now := time.Now()
	action.IsCompleted = true
	action.CompletedAt = &now

	if err := s.actionRepo.Update(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// UncompleteAction marks an action item as not completed
func (s *RetrospectiveService) UncompleteAction(ctx context.Context, id uuid.UUID) (*models.ActionItem, error) {
	action, err := s.actionRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrActionNotFound
		}
		return nil, err
	}

	action.IsCompleted = false
	action.CompletedAt = nil

	if err := s.actionRepo.Update(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// PatchAction partially updates an action item (status, assignee)
func (s *RetrospectiveService) PatchAction(ctx context.Context, id uuid.UUID, input PatchActionInput) (*models.ActionItem, error) {
	action, err := s.actionRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrActionNotFound
		}
		return nil, err
	}

	if input.Status != nil {
		action.Status = *input.Status
		if *input.Status == "done" {
			now := time.Now()
			action.IsCompleted = true
			action.CompletedAt = &now
		} else {
			action.IsCompleted = false
			action.CompletedAt = nil
		}
	}
	if input.AssigneeID != nil {
		action.AssigneeID = input.AssigneeID
	}
	if input.Description != nil {
		action.Description = input.Description
	}

	if err := s.actionRepo.Update(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// DeleteAction deletes an action item
func (s *RetrospectiveService) DeleteAction(ctx context.Context, id uuid.UUID) error {
	return s.actionRepo.Delete(ctx, id)
}

// ListActions lists action items for a retrospective
func (s *RetrospectiveService) ListActions(ctx context.Context, retroID uuid.UUID) ([]*models.ActionItem, error) {
	return s.actionRepo.ListByRetro(ctx, retroID)
}

// ListActionsByTeam lists all action items for a team's completed retrospectives
func (s *RetrospectiveService) ListActionsByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.ActionItem, error) {
	return s.actionRepo.ListByTeam(ctx, teamID)
}

// ListTemplates lists templates (built-in and team-specific)
func (s *RetrospectiveService) ListTemplates(ctx context.Context, teamID *uuid.UUID) ([]*models.Template, error) {
	if teamID != nil {
		return s.templateRepo.ListByTeam(ctx, *teamID)
	}
	return s.templateRepo.ListBuiltIn(ctx)
}

// GetTemplate gets a template by ID
func (s *RetrospectiveService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.Template, error) {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return template, nil
}

// CreateTemplate creates a new template
func (s *RetrospectiveService) CreateTemplate(ctx context.Context, template *models.Template) (*models.Template, error) {
	return s.templateRepo.Create(ctx, template)
}

// SetIcebreakerMood sets a user's mood in the icebreaker phase
func (s *RetrospectiveService) SetIcebreakerMood(ctx context.Context, retroID, userID uuid.UUID, mood models.MoodWeather) (*models.IcebreakerMood, error) {
	return s.icebreakerRepo.SetMood(ctx, retroID, userID, mood)
}

// GetIcebreakerMoods gets all moods for a retrospective
func (s *RetrospectiveService) GetIcebreakerMoods(ctx context.Context, retroID uuid.UUID) ([]*models.IcebreakerMood, error) {
	return s.icebreakerRepo.ListMoods(ctx, retroID)
}

// CountIcebreakerMoods counts moods for a retrospective
func (s *RetrospectiveService) CountIcebreakerMoods(ctx context.Context, retroID uuid.UUID) (int, error) {
	return s.icebreakerRepo.CountMoods(ctx, retroID)
}

// SetRotiVote sets a user's ROTI vote
func (s *RetrospectiveService) SetRotiVote(ctx context.Context, retroID, userID uuid.UUID, rating int) (*models.RotiVote, error) {
	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}
	return s.rotiRepo.SetVote(ctx, retroID, userID, rating)
}

// GetRotiResults gets the aggregated ROTI results
func (s *RetrospectiveService) GetRotiResults(ctx context.Context, retroID uuid.UUID) (*models.RotiResults, error) {
	results, err := s.rotiRepo.GetResults(ctx, retroID)
	if err != nil {
		return nil, err
	}

	// Include individual votes only if revealed
	if results.Revealed {
		votes, err := s.rotiRepo.ListVotes(ctx, retroID)
		if err != nil {
			return nil, err
		}
		results.Votes = votes
	}

	return results, nil
}

// RevealRotiResults reveals the ROTI results
func (s *RetrospectiveService) RevealRotiResults(ctx context.Context, retroID uuid.UUID) (*models.RotiResults, error) {
	if err := s.rotiRepo.RevealResults(ctx, retroID); err != nil {
		return nil, err
	}
	return s.GetRotiResults(ctx, retroID)
}

// CountRotiVotes counts ROTI votes for a retrospective
func (s *RetrospectiveService) CountRotiVotes(ctx context.Context, retroID uuid.UUID) (int, error) {
	return s.rotiRepo.CountVotes(ctx, retroID)
}
