package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/bus"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var (
	ErrNoActiveTimer = errors.New("no active timer")
	ErrTimerPaused   = errors.New("timer is paused")
)

// RetroTimer represents an active timer for a retrospective
type RetroTimer struct {
	RetroID          uuid.UUID
	Phase            models.RetroPhase
	Duration         time.Duration
	StartedAt        time.Time
	PausedAt         *time.Time
	RemainingAtPause time.Duration
	ticker           *time.Ticker
	done             chan struct{}
}

// Stop stops the timer
func (t *RetroTimer) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
	close(t.done)
}

// TimerService manages retrospective timers
type TimerService struct {
	bridge       bus.MessageBus
	retroRepo    *postgres.RetrospectiveRepository
	templateRepo *postgres.TemplateRepository
	timers       map[uuid.UUID]*RetroTimer
	mu           sync.RWMutex
}

// NewTimerService creates a new timer service
func NewTimerService(bridge bus.MessageBus, retroRepo *postgres.RetrospectiveRepository, templateRepo *postgres.TemplateRepository) *TimerService {
	return &TimerService{
		bridge:       bridge,
		retroRepo:    retroRepo,
		templateRepo: templateRepo,
		timers:       make(map[uuid.UUID]*RetroTimer),
	}
}

// StartTimer starts a timer for a retrospective
func (s *TimerService) StartTimer(ctx context.Context, retroID uuid.UUID, durationSec int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing timer if present
	if existing, ok := s.timers[retroID]; ok {
		existing.Stop()
		delete(s.timers, retroID)
	}

	retro, err := s.retroRepo.FindByID(ctx, retroID)
	if err != nil {
		return err
	}

	// Get default duration if not specified
	if durationSec <= 0 {
		durationSec, _ = s.getDefaultDuration(ctx, retro.TemplateID, retro.CurrentPhase)
	}

	now := time.Now()
	timer := &RetroTimer{
		RetroID:   retroID,
		Phase:     retro.CurrentPhase,
		Duration:  time.Duration(durationSec) * time.Second,
		StartedAt: now,
		done:      make(chan struct{}),
	}

	s.timers[retroID] = timer

	// Update database
	_ = s.retroRepo.UpdateTimer(ctx, retroID, &now, &durationSec, nil, nil)

	// Broadcast timer_started
	s.bridge.BroadcastToRoom(retroID.String(), websocket.Message{
		Type: "timer_started",
		Payload: map[string]interface{}{
			"phase":            timer.Phase,
			"duration_seconds": durationSec,
			"end_at":           timer.StartedAt.Add(timer.Duration).Format(time.RFC3339),
		},
	})

	// Start ticker goroutine
	go s.runTimer(timer)

	return nil
}

// runTimer runs the timer ticker
func (s *TimerService) runTimer(timer *RetroTimer) {
	timer.ticker = time.NewTicker(1 * time.Second)
	defer timer.ticker.Stop()

	for {
		select {
		case <-timer.done:
			return
		case <-timer.ticker.C:
			remaining := s.getRemainingTime(timer)

			// Broadcast tick every 5 seconds to reduce traffic
			if int(remaining.Seconds())%5 == 0 || remaining.Seconds() <= 10 {
				s.bridge.BroadcastToRoom(timer.RetroID.String(), websocket.Message{
					Type: "timer_tick",
					Payload: map[string]interface{}{
						"remaining_seconds": int(remaining.Seconds()),
						"phase":             timer.Phase,
					},
				})
			}

			// Timer ended
			if remaining <= 0 {
				s.bridge.BroadcastToRoom(timer.RetroID.String(), websocket.Message{
					Type: "timer_ended",
					Payload: map[string]interface{}{
						"phase": timer.Phase,
					},
				})
				s.mu.Lock()
				delete(s.timers, timer.RetroID)
				s.mu.Unlock()
				return
			}
		}
	}
}

// PauseTimer pauses a timer
func (s *TimerService) PauseTimer(ctx context.Context, retroID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return ErrNoActiveTimer
	}

	if timer.PausedAt != nil {
		return ErrTimerPaused
	}

	now := time.Now()
	timer.PausedAt = &now
	timer.RemainingAtPause = s.getRemainingTime(timer)
	timer.ticker.Stop()

	remaining := int(timer.RemainingAtPause.Seconds())
	_ = s.retroRepo.UpdateTimer(ctx, retroID, nil, nil, &now, &remaining)

	s.bridge.BroadcastToRoom(retroID.String(), websocket.Message{
		Type: "timer_paused",
		Payload: map[string]interface{}{
			"remaining_seconds": remaining,
		},
	})

	return nil
}

// ResumeTimer resumes a paused timer
func (s *TimerService) ResumeTimer(ctx context.Context, retroID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return ErrNoActiveTimer
	}

	if timer.PausedAt == nil {
		return nil // Not paused
	}

	// Calculate new start time to maintain remaining duration
	now := time.Now()
	timer.StartedAt = now.Add(-timer.Duration + timer.RemainingAtPause)
	timer.PausedAt = nil

	// Update database
	_ = s.retroRepo.UpdateTimer(ctx, retroID, &timer.StartedAt, nil, nil, nil)

	// Restart ticker
	go s.runTimer(timer)

	s.bridge.BroadcastToRoom(retroID.String(), websocket.Message{
		Type: "timer_resumed",
		Payload: map[string]interface{}{
			"remaining_seconds": int(timer.RemainingAtPause.Seconds()),
			"end_at":            timer.StartedAt.Add(timer.Duration).Format(time.RFC3339),
		},
	})

	return nil
}

// ResetTimer resets a timer to its original duration
func (s *TimerService) ResetTimer(ctx context.Context, retroID uuid.UUID) error {
	s.mu.Lock()
	timer, ok := s.timers[retroID]
	s.mu.Unlock()

	if !ok {
		return ErrNoActiveTimer
	}

	// Stop and restart
	timer.Stop()
	return s.StartTimer(ctx, retroID, int(timer.Duration.Seconds()))
}

// AddTime adds time to a running timer
func (s *TimerService) AddTime(ctx context.Context, retroID uuid.UUID, secondsToAdd int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return ErrNoActiveTimer
	}

	timer.Duration += time.Duration(secondsToAdd) * time.Second
	newRemaining := s.getRemainingTime(timer)

	durationSec := int(timer.Duration.Seconds())
	_ = s.retroRepo.UpdateTimer(ctx, retroID, nil, &durationSec, nil, nil)

	s.bridge.BroadcastToRoom(retroID.String(), websocket.Message{
		Type: "timer_extended",
		Payload: map[string]interface{}{
			"added_seconds":  secondsToAdd,
			"new_remaining":  int(newRemaining.Seconds()),
			"new_end_at":     timer.StartedAt.Add(timer.Duration).Format(time.RFC3339),
		},
	})

	return nil
}

// StopTimer stops a timer
func (s *TimerService) StopTimer(ctx context.Context, retroID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return nil
	}

	timer.Stop()
	delete(s.timers, retroID)

	// Clear database
	_ = s.retroRepo.UpdateTimer(ctx, retroID, nil, nil, nil, nil)

	return nil
}

// GetRemainingSeconds returns the remaining seconds for a timer
func (s *TimerService) GetRemainingSeconds(retroID uuid.UUID) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return 0
	}

	return int(s.getRemainingTime(timer).Seconds())
}

// IsTimerRunning checks if a timer is running
func (s *TimerService) IsTimerRunning(retroID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timer, ok := s.timers[retroID]
	if !ok {
		return false
	}

	return timer.PausedAt == nil
}

// getRemainingTime calculates remaining time for a timer
func (s *TimerService) getRemainingTime(timer *RetroTimer) time.Duration {
	if timer.PausedAt != nil {
		return timer.RemainingAtPause
	}
	elapsed := time.Since(timer.StartedAt)
	remaining := timer.Duration - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// getDefaultDuration gets the default duration for a phase
func (s *TimerService) getDefaultDuration(ctx context.Context, templateID uuid.UUID, phase models.RetroPhase) (int, error) {
	template, err := s.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		// Return defaults if template not found
		defaults := map[models.RetroPhase]int{
			models.PhaseBrainstorm: 300,
			models.PhaseGroup:      180,
			models.PhaseVote:       180,
			models.PhaseDiscuss:    900,
			models.PhaseAction:     300,
		}
		return defaults[phase], nil
	}

	if duration, ok := template.PhaseTimes[phase]; ok {
		return duration, nil
	}

	// Defaults
	return 300, nil
}
