package services

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var (
	ErrNoTopicsToDiscuss = errors.New("no topics to discuss")
	ErrSessionNotLC      = errors.New("session is not a lean coffee")
)

// LCDiscussionState represents the current state of a Lean Coffee discussion
type LCDiscussionState struct {
	CurrentTopicID *uuid.UUID             `json:"currentTopicId"`
	Queue          []*models.Item         `json:"queue"`
	Done           []*models.Item         `json:"done"`
	TopicHistory   []*models.LCTopicHistory `json:"topicHistory"`
}

// LeanCoffeeService handles Lean Coffee specific operations
type LeanCoffeeService struct {
	retroRepo        *postgres.RetrospectiveRepository
	itemRepo         *postgres.ItemRepository
	voteRepo         *postgres.VoteRepository
	topicHistoryRepo *postgres.LCTopicHistoryRepository
}

// NewLeanCoffeeService creates a new Lean Coffee service
func NewLeanCoffeeService(
	retroRepo *postgres.RetrospectiveRepository,
	itemRepo *postgres.ItemRepository,
	voteRepo *postgres.VoteRepository,
	topicHistoryRepo *postgres.LCTopicHistoryRepository,
) *LeanCoffeeService {
	return &LeanCoffeeService{
		retroRepo:        retroRepo,
		itemRepo:         itemRepo,
		voteRepo:         voteRepo,
		topicHistoryRepo: topicHistoryRepo,
	}
}

// NextTopic closes the current topic and moves to the next most-voted undiscussed topic.
// Returns the new topic history entry and the updated retro.
func (s *LeanCoffeeService) NextTopic(ctx context.Context, sessionID uuid.UUID) (*models.LCTopicHistory, *models.Retrospective, error) {
	retro, err := s.retroRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	if retro.SessionType != models.SessionTypeLeanCoffee {
		return nil, nil, ErrSessionNotLC
	}

	// Close current topic if there is one
	if retro.LCCurrentTopicID != nil {
		currentHistory, err := s.topicHistoryRepo.FindByTopic(ctx, sessionID, *retro.LCCurrentTopicID)
		if err == nil && currentHistory.EndedAt == nil {
			now := time.Now()
			currentHistory.EndedAt = &now
			elapsed := now.Sub(currentHistory.StartedAt)
			currentHistory.TotalDiscussionSeconds = int(elapsed.Seconds())
			if err := s.topicHistoryRepo.Update(ctx, currentHistory); err != nil {
				slog.Error("failed to close current topic history", "error", err)
			}
		}
	}

	// Get all items and their vote counts
	items, err := s.itemRepo.ListByRetro(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	// Get already discussed topic IDs
	histories, err := s.topicHistoryRepo.ListByRetro(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	discussedIDs := make(map[uuid.UUID]bool)
	for _, h := range histories {
		if h.EndedAt != nil {
			discussedIDs[h.TopicID] = true
		}
	}
	// Also exclude the current topic being closed
	if retro.LCCurrentTopicID != nil {
		discussedIDs[*retro.LCCurrentTopicID] = true
	}

	// Filter undiscussed items and sort by vote count descending
	var candidates []*models.Item
	for _, item := range items {
		if !discussedIDs[item.ID] {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		// No more topics - clear current topic
		retro.LCCurrentTopicID = nil
		if err := s.retroRepo.Update(ctx, retro); err != nil {
			return nil, nil, err
		}
		return nil, retro, ErrNoTopicsToDiscuss
	}

	// Sort by vote count descending, then by creation time ascending
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].VoteCount != candidates[j].VoteCount {
			return candidates[i].VoteCount > candidates[j].VoteCount
		}
		return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
	})

	nextTopic := candidates[0]

	// Create history entry for the new topic
	nextOrder, err := s.topicHistoryRepo.GetNextOrder(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	history := &models.LCTopicHistory{
		RetroID:         sessionID,
		TopicID:         nextTopic.ID,
		DiscussionOrder: nextOrder,
		StartedAt:       now,
	}

	history, err = s.topicHistoryRepo.Create(ctx, history)
	if err != nil {
		return nil, nil, err
	}

	// Update current topic on retro
	retro.LCCurrentTopicID = &nextTopic.ID
	if err := s.retroRepo.Update(ctx, retro); err != nil {
		return nil, nil, err
	}

	return history, retro, nil
}

// SetTopic sets a specific topic as the current discussion topic.
// Used by discuss_set_item message handler.
func (s *LeanCoffeeService) SetTopic(ctx context.Context, sessionID, topicID uuid.UUID) (*models.LCTopicHistory, *models.Retrospective, error) {
	retro, err := s.retroRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	if retro.SessionType != models.SessionTypeLeanCoffee {
		return nil, nil, ErrSessionNotLC
	}

	// Close current topic if there is one and it's different
	if retro.LCCurrentTopicID != nil && *retro.LCCurrentTopicID != topicID {
		currentHistory, err := s.topicHistoryRepo.FindByTopic(ctx, sessionID, *retro.LCCurrentTopicID)
		if err == nil && currentHistory.EndedAt == nil {
			now := time.Now()
			currentHistory.EndedAt = &now
			elapsed := now.Sub(currentHistory.StartedAt)
			currentHistory.TotalDiscussionSeconds = int(elapsed.Seconds())
			if err := s.topicHistoryRepo.Update(ctx, currentHistory); err != nil {
				slog.Error("failed to close current topic history", "error", err)
			}
		}
	}

	// Check if topic already has history (resuming discussion)
	history, err := s.topicHistoryRepo.FindByTopic(ctx, sessionID, topicID)
	if err != nil {
		if !errors.Is(err, postgres.ErrNotFound) {
			return nil, nil, err
		}
		// Create new history entry
		nextOrder, err := s.topicHistoryRepo.GetNextOrder(ctx, sessionID)
		if err != nil {
			return nil, nil, err
		}

		now := time.Now()
		history = &models.LCTopicHistory{
			RetroID:         sessionID,
			TopicID:         topicID,
			DiscussionOrder: nextOrder,
			StartedAt:       now,
		}
		history, err = s.topicHistoryRepo.Create(ctx, history)
		if err != nil {
			return nil, nil, err
		}
	}

	// Update current topic
	retro.LCCurrentTopicID = &topicID
	if err := s.retroRepo.Update(ctx, retro); err != nil {
		return nil, nil, err
	}

	return history, retro, nil
}

// GetDiscussionState returns the full discussion state for a Lean Coffee session
func (s *LeanCoffeeService) GetDiscussionState(ctx context.Context, sessionID uuid.UUID) (*LCDiscussionState, error) {
	retro, err := s.retroRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if retro.SessionType != models.SessionTypeLeanCoffee {
		return nil, ErrSessionNotLC
	}

	items, err := s.itemRepo.ListByRetro(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	histories, err := s.topicHistoryRepo.ListByRetro(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Build sets of discussed and currently discussing IDs
	discussedIDs := make(map[uuid.UUID]bool)
	for _, h := range histories {
		if h.EndedAt != nil {
			discussedIDs[h.TopicID] = true
		}
	}

	var queue []*models.Item
	var done []*models.Item

	for _, item := range items {
		if retro.LCCurrentTopicID != nil && item.ID == *retro.LCCurrentTopicID {
			continue // Skip current topic (it's neither in queue nor done)
		}
		if discussedIDs[item.ID] {
			done = append(done, item)
		} else {
			queue = append(queue, item)
		}
	}

	// Sort queue by vote count descending
	sort.Slice(queue, func(i, j int) bool {
		if queue[i].VoteCount != queue[j].VoteCount {
			return queue[i].VoteCount > queue[j].VoteCount
		}
		return queue[i].CreatedAt.Before(queue[j].CreatedAt)
	})

	// Sort done by discussion order
	doneOrderMap := make(map[uuid.UUID]int)
	for _, h := range histories {
		doneOrderMap[h.TopicID] = h.DiscussionOrder
	}
	sort.Slice(done, func(i, j int) bool {
		return doneOrderMap[done[i].ID] < doneOrderMap[done[j].ID]
	})

	return &LCDiscussionState{
		CurrentTopicID: retro.LCCurrentTopicID,
		Queue:          queue,
		Done:           done,
		TopicHistory:   histories,
	}, nil
}

// GetTopicHistory returns the discussion history for a session
func (s *LeanCoffeeService) GetTopicHistory(ctx context.Context, sessionID uuid.UUID) ([]*models.LCTopicHistory, error) {
	return s.topicHistoryRepo.ListByRetro(ctx, sessionID)
}

// ListTopicsByTeam lists all discussed topics for a team
func (s *LeanCoffeeService) ListTopicsByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.DiscussedTopic, error) {
	return s.topicHistoryRepo.ListByTeam(ctx, teamID)
}
