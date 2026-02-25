package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/jycamier/retrotro/backend/internal/bus"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/services"
	ws "github.com/jycamier/retrotro/backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin check in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub               *ws.Hub
	bridge            bus.MessageBus
	retroService      *services.RetrospectiveService
	timerService      *services.TimerService
	authService       *services.AuthService
	leanCoffeeService *services.LeanCoffeeService
	teamMemberRepo    TeamMemberRepository
	attendeeRepo      AttendeeRepository
}

// TeamMemberRepository interface for team member operations
type TeamMemberRepository interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.TeamMember, error)
	GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*models.TeamMember, error)
}

// AttendeeRepository interface for attendance operations
type AttendeeRepository interface {
	Record(ctx context.Context, retroID, userID uuid.UUID, attended bool) error
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	hub *ws.Hub,
	bridge bus.MessageBus,
	retroService *services.RetrospectiveService,
	timerService *services.TimerService,
	authService *services.AuthService,
	leanCoffeeService *services.LeanCoffeeService,
	teamMemberRepo TeamMemberRepository,
	attendeeRepo AttendeeRepository,
) *WebSocketHandler {
	h := &WebSocketHandler{
		hub:               hub,
		bridge:            bridge,
		retroService:      retroService,
		timerService:      timerService,
		authService:       authService,
		leanCoffeeService: leanCoffeeService,
		teamMemberRepo:    teamMemberRepo,
		attendeeRepo:      attendeeRepo,
	}

	// Set callback for when user leaves room (handles abrupt browser close via grace period)
	hub.OnUserLeftRoom = func(roomID string, userID uuid.UUID) {
		// Publish presence leave to other pods
		bridge.PublishPresenceLeave(roomID, userID)
		// Relay participant_left to remote pods (local broadcast already done by Hub)
		bridge.PublishToRemotePods(roomID, ws.Message{
			Type: "participant_left",
			Payload: map[string]interface{}{
				"userId": userID,
			},
		})
		slog.Debug("OnUserLeftRoom callback triggered",
			"roomId", roomID,
			"userId", userID.String(),
		)
		retroID, err := uuid.Parse(roomID)
		if err != nil {
			slog.Debug("OnUserLeftRoom: failed to parse roomID", "error", err)
			return
		}
		retro, err := retroService.GetByID(context.Background(), retroID)
		if err != nil {
			slog.Debug("OnUserLeftRoom: failed to get retro", "error", err)
			return
		}
		slog.Debug("OnUserLeftRoom: checking phase",
			"currentPhase", retro.CurrentPhase,
			"isWaiting", retro.CurrentPhase == models.PhaseWaiting,
		)
		// Only broadcast team status update during waiting phase
		if retro.CurrentPhase == models.PhaseWaiting {
			slog.Debug("OnUserLeftRoom: broadcasting team members status")
			h.broadcastTeamMembersStatus(retroID, retro.TeamID)
		}
	}

	return h
}

// WSMessage represents an incoming WebSocket message
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// HandleConnection handles a new WebSocket connection
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "invalid token claims", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	client := &ws.Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		UserName: claims.Name,
		Hub:      h.hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	// Register client
	h.hub.Register(client)

	// Start goroutines
	go client.WritePump()
	go client.ReadPump(h.handleMessage)
}

// handleMessage handles incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(client *ws.Client, data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	log.Printf("Received WebSocket message type: %s", msg.Type)

	switch msg.Type {
	case "join_retro":
		h.handleJoinRetro(client, msg.Payload)
	case "leave_retro":
		h.handleLeaveRetro(client)
	case "heartbeat":
		// No-op: client sending heartbeat to keep connection alive
		// Useful for detecting stale connections and keeping connection active on high-latency networks
		slog.Debug("received heartbeat", "userId", client.UserID.String())
	case "item_create":
		h.handleItemCreate(client, msg.Payload)
	case "item_update":
		h.handleItemUpdate(client, msg.Payload)
	case "item_delete":
		h.handleItemDelete(client, msg.Payload)
	case "item_group":
		h.handleItemGroup(client, msg.Payload)
	case "vote_add":
		h.handleVoteAdd(client, msg.Payload)
	case "vote_remove":
		h.handleVoteRemove(client, msg.Payload)
	case "timer_start":
		h.handleTimerStart(client, msg.Payload)
	case "timer_pause":
		h.handleTimerPause(client)
	case "timer_resume":
		h.handleTimerResume(client)
	case "timer_add_time":
		h.handleTimerAddTime(client, msg.Payload)
	case "phase_next":
		h.handlePhaseNext(client)
	case "phase_set":
		h.handlePhaseSet(client, msg.Payload)
	case "action_create":
		h.handleActionCreate(client, msg.Payload)
	case "action_complete":
		h.handleActionComplete(client, msg.Payload)
	case "action_uncomplete":
		h.handleActionUncomplete(client, msg.Payload)
	case "action_delete":
		h.handleActionDelete(client, msg.Payload)
	case "retro_end":
		h.handleRetroEnd(client)
	case "mood_set":
		h.handleMoodSet(client, msg.Payload)
	case "roti_vote":
		h.handleRotiVote(client, msg.Payload)
	case "roti_reveal":
		h.handleRotiReveal(client)
	case "draft_typing":
		h.handleDraftTyping(client, msg.Payload)
	case "draft_clear":
		h.handleDraftClear(client, msg.Payload)
	case "facilitator_claim":
		h.handleFacilitatorClaim(client)
	case "facilitator_transfer":
		h.handleFacilitatorTransfer(client, msg.Payload)
	case "discuss_set_item":
		h.handleDiscussSetItem(client, msg.Payload)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleJoinRetro handles joining a retrospective room
func (h *WebSocketHandler) handleJoinRetro(client *ws.Client, payload json.RawMessage) {
	var data struct {
		RetroID string `json:"retroId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"code":    "invalid_payload",
				"message": "Invalid join request payload",
			},
		})
		return
	}

	retroID, err := uuid.Parse(data.RetroID)
	if err != nil {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"code":    "invalid_retro_id",
				"message": "Invalid retrospective ID",
			},
		})
		return
	}

	// Check if user already in room (to avoid duplicate join broadcasts)
	userAlreadyInRoom := h.hub.IsUserInRoom(retroID.String(), client.UserID)
	slog.Debug("user joining retro",
		"retroId", retroID.String(),
		"userId", client.UserID.String(),
		"userName", client.UserName,
		"alreadyInRoom", userAlreadyInRoom,
	)

	// Join room
	h.hub.JoinRoom(client, retroID.String())

	// Send current retro state
	retro, err := h.retroService.GetByID(context.Background(), retroID)
	if err != nil {
		slog.Error("failed to get retro for join",
			"retroId", retroID.String(),
			"userId", client.UserID.String(),
			"error", err,
		)
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"code":    "join_failed",
				"message": "Failed to join retrospective. Please try again.",
			},
		})
		return
	}

	items, _ := h.retroService.ListItems(context.Background(), retroID)
	actions, _ := h.retroService.ListActions(context.Background(), retroID)
	moods, _ := h.retroService.GetIcebreakerMoods(context.Background(), retroID)
	rotiResults, _ := h.retroService.GetRotiResults(context.Background(), retroID)
	voteSummary, _ := h.retroService.GetVoteSummary(context.Background(), retroID)

	// Get participants (currently connected, local + remote)
	participants := h.bridge.GetRoomClients(retroID.String())
	participantList := make([]map[string]interface{}, len(participants))
	connectedUserIds := make(map[uuid.UUID]bool)
	for i, p := range participants {
		participantList[i] = map[string]interface{}{
			"userId": p.UserID,
			"name":   p.UserName,
		}
		connectedUserIds[p.UserID] = true
	}

	// Get team members with connection status (for waiting room)
	var teamMembersWithStatus []models.TeamMemberStatus
	if retro.CurrentPhase == models.PhaseWaiting {
		teamMembers, err := h.teamMemberRepo.ListByTeam(context.Background(), retro.TeamID)
		if err == nil {
			teamMembersWithStatus = make([]models.TeamMemberStatus, len(teamMembers))
			for i, member := range teamMembers {
				isConnected := connectedUserIds[member.UserID]
				slog.Debug("building team member status for retro_state",
					"memberId", member.UserID.String(),
					"memberName", member.User.DisplayName,
					"isConnected", isConnected,
				)
				teamMembersWithStatus[i] = models.TeamMemberStatus{
					UserID:      member.UserID,
					DisplayName: member.User.DisplayName,
					AvatarURL:   member.User.AvatarURL,
					Role:        member.Role,
					IsConnected: isConnected,
				}
			}
		}
	}

	// Convert voteSummary to JSON-friendly format with string keys
	voteSummaryJSON := make(map[string]map[string]int)
	for userID, itemVotes := range voteSummary {
		userKey := userID.String()
		voteSummaryJSON[userKey] = make(map[string]int)
		for itemID, count := range itemVotes {
			voteSummaryJSON[userKey][itemID.String()] = count
		}
	}

	// Build retro_state payload
	retroStatePayload := map[string]interface{}{
		"retro":          retro,
		"items":          items,
		"actions":        actions,
		"participants":   participantList,
		"timerRunning":   h.timerService.IsTimerRunning(retroID),
		"timerRemaining": h.timerService.GetRemainingSeconds(retroID),
		"moods":          moods,
		"rotiResults":    rotiResults,
		"teamMembers":    teamMembersWithStatus,
		"voteSummary":    voteSummaryJSON,
	}

	// Add LC discussion state if this is a Lean Coffee session
	if retro.SessionType == models.SessionTypeLeanCoffee {
		lcState, err := h.leanCoffeeService.GetDiscussionState(context.Background(), retroID)
		if err == nil {
			retroStatePayload["lcDiscussionState"] = lcState
		}
	}

	h.hub.SendToClient(client, ws.Message{
		Type:    "retro_state",
		Payload: retroStatePayload,
	})

	// Broadcast participant joined only if user wasn't already in room (local check only)
	if !userAlreadyInRoom {
		h.bridge.BroadcastToRoomExcept(retroID.String(), ws.Message{
			Type: "participant_joined",
			Payload: map[string]interface{}{
				"userId": client.UserID,
				"name":   client.UserName,
			},
		}, client)

		// Publish presence join to other pods
		h.bridge.PublishPresenceJoin(retroID.String(), client.UserID, client.UserName)

		// Broadcast team member status update if in waiting phase
		slog.Debug("checking if should broadcast team status",
			"retroId", retroID.String(),
			"currentPhase", retro.CurrentPhase,
			"isWaiting", retro.CurrentPhase == models.PhaseWaiting,
		)
		if retro.CurrentPhase == models.PhaseWaiting {
			h.broadcastTeamMembersStatus(retroID, retro.TeamID)
		}
	}
}

// broadcastTeamMembersStatus broadcasts the updated team members status to all clients in the room
func (h *WebSocketHandler) broadcastTeamMembersStatus(retroID, teamID uuid.UUID) {
	// Get current participants (local + remote)
	participants := h.bridge.GetRoomClients(retroID.String())
	connectedUserIds := make(map[uuid.UUID]bool)
	slog.Debug("broadcast team members status",
		"retroId", retroID.String(),
		"connectedClientsCount", len(participants),
	)
	for _, p := range participants {
		slog.Debug("connected client in room",
			"retroId", retroID.String(),
			"userId", p.UserID.String(),
			"userName", p.UserName,
		)
		connectedUserIds[p.UserID] = true
	}

	// Get team members with status
	teamMembers, err := h.teamMemberRepo.ListByTeam(context.Background(), teamID)
	if err != nil {
		log.Printf("Failed to get team members: %v", err)
		return
	}

	teamMembersWithStatus := make([]models.TeamMemberStatus, len(teamMembers))
	for i, member := range teamMembers {
		teamMembersWithStatus[i] = models.TeamMemberStatus{
			UserID:      member.UserID,
			DisplayName: member.User.DisplayName,
			AvatarURL:   member.User.AvatarURL,
			Role:        member.Role,
			IsConnected: connectedUserIds[member.UserID],
		}
	}

	h.bridge.BroadcastToRoom(retroID.String(), ws.Message{
		Type: "team_members_updated",
		Payload: map[string]interface{}{
			"teamMembers": teamMembersWithStatus,
		},
	})
}

// handleLeaveRetro handles leaving a retrospective room
func (h *WebSocketHandler) handleLeaveRetro(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	roomID := client.RoomID
	userID := client.UserID

	slog.Debug("user leaving retro",
		"retroId", roomID,
		"userId", userID.String(),
		"userName", client.UserName,
	)

	// Get retro info before leaving to check if we need to broadcast team member status
	retroID, err := uuid.Parse(roomID)
	var retro *models.Retrospective
	if err == nil {
		retro, _ = h.retroService.GetByID(context.Background(), retroID)
	}

	h.hub.LeaveRoom(client)

	// Only broadcast participant_left if user has no more local connections in room
	if !h.hub.IsUserInRoom(roomID, userID) {
		h.bridge.BroadcastToRoom(roomID, ws.Message{
			Type: "participant_left",
			Payload: map[string]interface{}{
				"userId": userID,
			},
		})

		// Publish presence leave to other pods
		h.bridge.PublishPresenceLeave(roomID, userID)

		// Broadcast team member status update if in waiting phase
		if retro != nil && retro.CurrentPhase == models.PhaseWaiting {
			h.broadcastTeamMembersStatus(retroID, retro.TeamID)
		}
	}
}

// handleItemCreate handles creating an item
func (h *WebSocketHandler) handleItemCreate(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		slog.Debug("handleItemCreate: client not in a room")
		return
	}

	var data struct {
		ColumnID string `json:"columnId"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		slog.Error("handleItemCreate: failed to unmarshal payload", "error", err)
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		slog.Error("handleItemCreate: invalid retroID", "retroID", client.RoomID, "error", err)
		return
	}

	slog.Info("handleItemCreate: creating item",
		"retroID", retroID.String(),
		"userID", client.UserID.String(),
		"columnID", data.ColumnID,
		"contentLength", len(data.Content),
	)

	item, err := h.retroService.CreateItem(context.Background(), retroID, client.UserID, services.CreateItemInput{
		ColumnID: data.ColumnID,
		Content:  data.Content,
	})
	if err != nil {
		slog.Error("handleItemCreate: failed to create item", "error", err)
		return
	}

	slog.Info("handleItemCreate: broadcasting item_created",
		"itemID", item.ID,
		"roomID", client.RoomID,
	)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "item_created",
		Payload: item,
	})
}

// handleItemUpdate handles updating an item
func (h *WebSocketHandler) handleItemUpdate(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ItemID  string `json:"itemId"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	itemID, err := uuid.Parse(data.ItemID)
	if err != nil {
		return
	}

	item, err := h.retroService.UpdateItem(context.Background(), itemID, data.Content)
	if err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "item_updated",
		Payload: item,
	})
}

// handleItemDelete handles deleting an item
func (h *WebSocketHandler) handleItemDelete(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ItemID string `json:"itemId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	itemID, err := uuid.Parse(data.ItemID)
	if err != nil {
		return
	}

	if err := h.retroService.DeleteItem(context.Background(), itemID); err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "item_deleted",
		Payload: map[string]interface{}{
			"itemId": data.ItemID,
		},
	})
}

// handleItemGroup handles grouping items together
func (h *WebSocketHandler) handleItemGroup(client *ws.Client, payload json.RawMessage) {
	log.Printf("handleItemGroup called, roomID: %s, payload: %s", client.RoomID, string(payload))

	if client.RoomID == "" {
		log.Printf("handleItemGroup: client not in a room")
		return
	}

	var data struct {
		ParentID string   `json:"parentId"`
		ChildIDs []string `json:"childIds"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleItemGroup: failed to unmarshal payload: %v", err)
		return
	}
	log.Printf("handleItemGroup: parentID=%s, childIDs=%v", data.ParentID, data.ChildIDs)

	parentID, err := uuid.Parse(data.ParentID)
	if err != nil {
		return
	}

	childIDs := make([]uuid.UUID, 0, len(data.ChildIDs))
	for _, idStr := range data.ChildIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		childIDs = append(childIDs, id)
	}

	allAffected, err := h.retroService.GroupItems(context.Background(), parentID, childIDs)
	if err != nil {
		log.Printf("handleItemGroup: GroupItems failed: %v", err)
		return
	}

	// Broadcast all affected IDs (including grandchildren moved to new parent)
	affectedStrings := make([]string, 0, len(allAffected))
	for _, id := range allAffected {
		affectedStrings = append(affectedStrings, id.String())
	}
	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "items_grouped",
		Payload: map[string]interface{}{
			"parentId": data.ParentID,
			"childIds": affectedStrings,
		},
	})
}

// handleVoteAdd handles adding a vote
func (h *WebSocketHandler) handleVoteAdd(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ItemID string `json:"itemId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	itemID, err := uuid.Parse(data.ItemID)
	if err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	if err := h.retroService.Vote(context.Background(), retroID, itemID, client.UserID); err != nil {
		if errors.Is(err, services.ErrVoteLimitReached) {
			h.hub.SendToClient(client, ws.Message{
				Type: "error",
				Payload: map[string]interface{}{
					"code":    "vote_limit_reached",
					"message": "Vous avez atteint la limite de votes",
				},
			})
		} else if errors.Is(err, services.ErrItemVoteLimitReached) {
			h.hub.SendToClient(client, ws.Message{
				Type: "error",
				Payload: map[string]interface{}{
					"code":    "item_vote_limit_reached",
					"message": "Limite de votes atteinte pour cet item",
				},
			})
		}
		return
	}

	// Get updated vote count for this user
	userVoteCount, _ := h.retroService.GetUserVoteCount(context.Background(), retroID, client.UserID)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "vote_updated",
		Payload: map[string]interface{}{
			"itemId":        data.ItemID,
			"action":        "add",
			"userId":        client.UserID,
			"userVoteCount": userVoteCount,
		},
	})
}

// handleVoteRemove handles removing a vote
func (h *WebSocketHandler) handleVoteRemove(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ItemID string `json:"itemId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	itemID, err := uuid.Parse(data.ItemID)
	if err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	if err := h.retroService.Unvote(context.Background(), itemID, client.UserID); err != nil {
		return
	}

	// Get updated vote count for this user
	userVoteCount, _ := h.retroService.GetUserVoteCount(context.Background(), retroID, client.UserID)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "vote_updated",
		Payload: map[string]interface{}{
			"itemId":        data.ItemID,
			"action":        "remove",
			"userId":        client.UserID,
			"userVoteCount": userVoteCount,
		},
	})
}

// handleTimerStart handles starting the timer
func (h *WebSocketHandler) handleTimerStart(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		DurationSeconds int `json:"duration_seconds"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	_ = h.timerService.StartTimer(context.Background(), retroID, data.DurationSeconds)
}

// handleTimerPause handles pausing the timer
func (h *WebSocketHandler) handleTimerPause(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	_ = h.timerService.PauseTimer(context.Background(), retroID)
}

// handleTimerResume handles resuming the timer
func (h *WebSocketHandler) handleTimerResume(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	_ = h.timerService.ResumeTimer(context.Background(), retroID)
}

// handleTimerAddTime handles adding time to the timer
func (h *WebSocketHandler) handleTimerAddTime(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		Seconds int `json:"seconds"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	_ = h.timerService.AddTime(context.Background(), retroID, data.Seconds)
}

// handlePhaseNext handles advancing to the next phase
func (h *WebSocketHandler) handlePhaseNext(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	ctx := context.Background()
	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		return
	}

	// Check if the client is the facilitator
	if retro.FacilitatorID != client.UserID {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Only the facilitator can change the phase",
			},
		})
		return
	}

	previousPhase := retro.CurrentPhase

	// If transitioning from waiting to icebreaker, record attendance
	if previousPhase == models.PhaseWaiting {
		teamMembers, err := h.teamMemberRepo.ListByTeam(ctx, retro.TeamID)
		if err == nil {
			// Get connected users (local + remote)
			participants := h.bridge.GetRoomClients(retroID.String())
			connectedUserIds := make(map[uuid.UUID]bool)
			for _, p := range participants {
				connectedUserIds[p.UserID] = true
			}

			// Record attendance for each team member
			for _, member := range teamMembers {
				_ = h.attendeeRepo.Record(ctx, retroID, member.UserID, connectedUserIds[member.UserID])
			}
		}
	}

	nextPhase, err := h.retroService.NextPhase(ctx, retroID)
	if err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "phase_changed",
		Payload: map[string]interface{}{
			"previous_phase": previousPhase,
			"current_phase":  nextPhase,
		},
	})

	// Auto-start timer for the new phase if configured
	h.autoStartPhaseTimer(ctx, retroID, retro.TemplateID, nextPhase)
}

// handlePhaseSet handles setting a specific phase
func (h *WebSocketHandler) handlePhaseSet(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		Phase string `json:"phase"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	ctx := context.Background()
	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		return
	}

	// Check if the client is the facilitator
	if retro.FacilitatorID != client.UserID {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Only the facilitator can change the phase",
			},
		})
		return
	}

	previousPhase := retro.CurrentPhase

	newPhase := models.RetroPhase(data.Phase)
	if err := h.retroService.SetPhase(ctx, retroID, newPhase); err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "phase_changed",
		Payload: map[string]interface{}{
			"previous_phase": previousPhase,
			"current_phase":  data.Phase,
		},
	})

	// Auto-start timer for the new phase if configured
	h.autoStartPhaseTimer(ctx, retroID, retro.TemplateID, newPhase)
}

// autoStartPhaseTimer starts the timer for a phase if a duration is configured
func (h *WebSocketHandler) autoStartPhaseTimer(ctx context.Context, retroID, templateID uuid.UUID, phase models.RetroPhase) {
	// Get the configured duration for this phase
	duration, err := h.retroService.GetPhaseDuration(ctx, templateID, phase)
	if err != nil {
		slog.Error("failed to get phase duration", "error", err)
		return
	}

	// Only start timer if duration is configured (> 0)
	if duration > 0 {
		if err := h.timerService.StartTimer(ctx, retroID, duration); err != nil {
			slog.Error("failed to auto-start timer", "error", err, "phase", phase)
		} else {
			slog.Info("auto-started timer", "retroId", retroID, "phase", phase, "duration", duration)
		}
	}
}

// handleActionCreate handles creating an action item
func (h *WebSocketHandler) handleActionCreate(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		Title      string  `json:"title"`
		AssigneeID *string `json:"assigneeId"`
		DueDate    *string `json:"dueDate"`
		ItemID     *string `json:"itemId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	input := services.CreateActionInput{
		Title: data.Title,
	}

	if data.AssigneeID != nil && *data.AssigneeID != "" {
		assigneeID, err := uuid.Parse(*data.AssigneeID)
		if err == nil {
			input.AssigneeID = &assigneeID
		}
	}

	if data.ItemID != nil && *data.ItemID != "" {
		itemID, err := uuid.Parse(*data.ItemID)
		if err == nil {
			input.ItemID = &itemID
		}
	}

	action, err := h.retroService.CreateAction(context.Background(), retroID, client.UserID, input)
	if err != nil {
		log.Printf("handleActionCreate: failed to create action: %v", err)
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "action_created",
		Payload: action,
	})
}

// handleActionComplete handles marking an action as completed
func (h *WebSocketHandler) handleActionComplete(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ActionID string `json:"actionId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	actionID, err := uuid.Parse(data.ActionID)
	if err != nil {
		return
	}

	action, err := h.retroService.CompleteAction(context.Background(), actionID)
	if err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "action_updated",
		Payload: action,
	})
}

// handleActionUncomplete handles marking an action as not completed
func (h *WebSocketHandler) handleActionUncomplete(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ActionID string `json:"actionId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	actionID, err := uuid.Parse(data.ActionID)
	if err != nil {
		return
	}

	action, err := h.retroService.UncompleteAction(context.Background(), actionID)
	if err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "action_updated",
		Payload: action,
	})
}

// handleActionDelete handles deleting an action item
func (h *WebSocketHandler) handleActionDelete(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ActionID string `json:"actionId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}

	actionID, err := uuid.Parse(data.ActionID)
	if err != nil {
		return
	}

	if err := h.retroService.DeleteAction(context.Background(), actionID); err != nil {
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "action_deleted",
		Payload: map[string]interface{}{
			"actionId": data.ActionID,
		},
	})
}

// handleRetroEnd handles ending a retrospective
func (h *WebSocketHandler) handleRetroEnd(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	retro, err := h.retroService.End(context.Background(), retroID)
	if err != nil {
		log.Printf("handleRetroEnd: failed to end retro: %v", err)
		return
	}

	// Get final items and actions for the summary
	items, _ := h.retroService.ListItems(context.Background(), retroID)
	actions, _ := h.retroService.ListActions(context.Background(), retroID)
	rotiResults, _ := h.retroService.GetRotiResults(context.Background(), retroID)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "retro_ended",
		Payload: map[string]interface{}{
			"retro":       retro,
			"items":       items,
			"actions":     actions,
			"rotiResults": rotiResults,
		},
	})
}

// handleMoodSet handles setting a user's mood in the icebreaker phase
func (h *WebSocketHandler) handleMoodSet(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		Mood string `json:"mood"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleMoodSet: failed to unmarshal payload: %v", err)
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	mood, err := h.retroService.SetIcebreakerMood(context.Background(), retroID, client.UserID, models.MoodWeather(data.Mood))
	if err != nil {
		log.Printf("handleMoodSet: failed to set mood: %v", err)
		return
	}

	// Get participant count and mood count
	participants := h.bridge.GetRoomClients(retroID.String())
	moodCount, _ := h.retroService.CountIcebreakerMoods(context.Background(), retroID)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "mood_updated",
		Payload: map[string]interface{}{
			"userId":           client.UserID,
			"userName":         client.UserName,
			"mood":             mood.Mood,
			"moodCount":        moodCount,
			"participantCount": len(participants),
		},
	})
}

// handleRotiVote handles a user's ROTI vote
func (h *WebSocketHandler) handleRotiVote(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		Rating int `json:"rating"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleRotiVote: failed to unmarshal payload: %v", err)
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	_, err = h.retroService.SetRotiVote(context.Background(), retroID, client.UserID, data.Rating)
	if err != nil {
		log.Printf("handleRotiVote: failed to set vote: %v", err)
		return
	}

	// Get participant count and vote count
	participants := h.bridge.GetRoomClients(retroID.String())
	voteCount, _ := h.retroService.CountRotiVotes(context.Background(), retroID)

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "roti_vote_submitted",
		Payload: map[string]interface{}{
			"userId":           client.UserID,
			"voteCount":        voteCount,
			"participantCount": len(participants),
		},
	})
}

// handleRotiReveal handles revealing ROTI results (facilitator only)
func (h *WebSocketHandler) handleRotiReveal(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	results, err := h.retroService.RevealRotiResults(context.Background(), retroID)
	if err != nil {
		log.Printf("handleRotiReveal: failed to reveal results: %v", err)
		return
	}

	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type:    "roti_results_revealed",
		Payload: results,
	})
}

// handleDraftTyping handles broadcasting draft typing status to other participants
func (h *WebSocketHandler) handleDraftTyping(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ColumnID      string `json:"columnId"`
		ContentLength int    `json:"contentLength"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleDraftTyping: failed to unmarshal payload: %v", err)
		return
	}

	// Broadcast to other users (not the author) that someone is typing
	h.bridge.BroadcastToRoomExcept(client.RoomID, ws.Message{
		Type: "draft_typing",
		Payload: map[string]interface{}{
			"userId":        client.UserID,
			"userName":      client.UserName,
			"columnId":      data.ColumnID,
			"contentLength": data.ContentLength,
		},
	}, client)
}

// handleDraftClear handles clearing a draft when user submits or clears the input
func (h *WebSocketHandler) handleDraftClear(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		ColumnID string `json:"columnId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleDraftClear: failed to unmarshal payload: %v", err)
		return
	}

	// Broadcast to other users that the draft is cleared
	h.bridge.BroadcastToRoomExcept(client.RoomID, ws.Message{
		Type: "draft_cleared",
		Payload: map[string]interface{}{
			"userId":   client.UserID,
			"columnId": data.ColumnID,
		},
	}, client)
}

// handleFacilitatorClaim handles a user claiming the facilitator role
func (h *WebSocketHandler) handleFacilitatorClaim(client *ws.Client) {
	if client.RoomID == "" {
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	ctx := context.Background()
	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		log.Printf("handleFacilitatorClaim: failed to get retro: %v", err)
		return
	}

	// Only allow claiming during waiting phase
	if retro.CurrentPhase != models.PhaseWaiting {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Facilitator can only be changed during the waiting phase",
			},
		})
		return
	}

	// Check if user has the right role (admin or facilitator of the team)
	member, err := h.teamMemberRepo.GetByTeamAndUser(ctx, retro.TeamID, client.UserID)
	if err != nil {
		log.Printf("handleFacilitatorClaim: failed to get team member: %v", err)
		return
	}

	if member.Role != models.RoleAdmin {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Only admins can claim the facilitator role",
			},
		})
		return
	}

	// Update the facilitator
	retro.FacilitatorID = client.UserID
	if err := h.retroService.Update(ctx, retro); err != nil {
		log.Printf("handleFacilitatorClaim: failed to update retro: %v", err)
		return
	}

	// Broadcast the change to all participants
	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "facilitator_changed",
		Payload: map[string]interface{}{
			"facilitatorId":   client.UserID,
			"facilitatorName": client.UserName,
		},
	})
}

// handleFacilitatorTransfer handles transferring the facilitator role to another participant
func (h *WebSocketHandler) handleFacilitatorTransfer(client *ws.Client, payload json.RawMessage) {
	if client.RoomID == "" {
		return
	}

	var data struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("handleFacilitatorTransfer: failed to unmarshal payload: %v", err)
		return
	}

	targetUserID, err := uuid.Parse(data.UserID)
	if err != nil {
		log.Printf("handleFacilitatorTransfer: invalid user ID: %v", err)
		return
	}

	retroID, err := uuid.Parse(client.RoomID)
	if err != nil {
		return
	}

	ctx := context.Background()
	retro, err := h.retroService.GetByID(ctx, retroID)
	if err != nil {
		log.Printf("handleFacilitatorTransfer: failed to get retro: %v", err)
		return
	}

	// Only allow transfer during waiting phase
	if retro.CurrentPhase != models.PhaseWaiting {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Facilitator can only be changed during the waiting phase",
			},
		})
		return
	}

	// Check if client is the current facilitator
	if retro.FacilitatorID != client.UserID {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Only the current facilitator can transfer the role",
			},
		})
		return
	}

	// Check if target user is in the room (local + remote)
	if !h.bridge.IsUserInRoom(client.RoomID, targetUserID) {
		h.hub.SendToClient(client, ws.Message{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Target user is not in the room",
			},
		})
		return
	}

	// Get target user name
	participants := h.bridge.GetRoomClients(client.RoomID)
	var targetUserName string
	for _, p := range participants {
		if p.UserID == targetUserID {
			targetUserName = p.UserName
			break
		}
	}

	// Update the facilitator
	retro.FacilitatorID = targetUserID
	if err := h.retroService.Update(ctx, retro); err != nil {
		log.Printf("handleFacilitatorTransfer: failed to update retro: %v", err)
		return
	}

	// Broadcast the change to all participants
	h.bridge.BroadcastToRoom(client.RoomID, ws.Message{
		Type: "facilitator_changed",
		Payload: map[string]interface{}{
			"facilitatorId":   targetUserID,
			"facilitatorName": targetUserName,
		},
	})
}
