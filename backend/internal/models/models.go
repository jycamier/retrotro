package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents user roles in teams
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// RetroPhase represents phases of a retrospective
type RetroPhase string

const (
	PhaseWaiting    RetroPhase = "waiting"
	PhaseIcebreaker RetroPhase = "icebreaker"
	PhaseBrainstorm RetroPhase = "brainstorm"
	PhaseGroup      RetroPhase = "group"
	PhaseVote       RetroPhase = "vote"
	PhaseDiscuss    RetroPhase = "discuss"
	PhaseAction     RetroPhase = "action"
	PhaseRoti       RetroPhase = "roti"
)

// MoodWeather represents weather-based mood for icebreaker
type MoodWeather string

const (
	MoodSunny        MoodWeather = "sunny"
	MoodPartlyCloudy MoodWeather = "partly_cloudy"
	MoodCloudy       MoodWeather = "cloudy"
	MoodRainy        MoodWeather = "rainy"
	MoodStormy       MoodWeather = "stormy"
)

// RetroStatus represents status of a retrospective
type RetroStatus string

const (
	StatusDraft     RetroStatus = "draft"
	StatusActive    RetroStatus = "active"
	StatusCompleted RetroStatus = "completed"
	StatusArchived  RetroStatus = "archived"
)

// User represents a user in the system
type User struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Email       string     `json:"email" db:"email"`
	DisplayName string     `json:"displayName" db:"display_name"`
	AvatarURL   *string    `json:"avatarUrl,omitempty" db:"avatar_url"`
	OIDCSubject string     `json:"-" db:"oidc_subject"`
	OIDCIssuer  string     `json:"-" db:"oidc_issuer"`
	IsAdmin     bool       `json:"isAdmin" db:"is_admin"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

// Team represents a team/group in the system
type Team struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Slug          string     `json:"slug" db:"slug"`
	Description   *string    `json:"description,omitempty" db:"description"`
	OIDCGroupID   *string    `json:"-" db:"oidc_group_id"`
	IsOIDCManaged bool       `json:"isOidcManaged" db:"is_oidc_managed"`
	CreatedBy     *uuid.UUID `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// TeamMember represents membership in a team
type TeamMember struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TeamID       uuid.UUID  `json:"teamId" db:"team_id"`
	UserID       uuid.UUID  `json:"userId" db:"user_id"`
	Role         Role       `json:"role" db:"role"`
	IsOIDCSynced bool       `json:"isOidcSynced" db:"is_oidc_synced"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty" db:"last_synced_at"`
	JoinedAt     time.Time  `json:"joinedAt" db:"joined_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
	Team *Team `json:"team,omitempty"`
}

// Template represents a retrospective template
type Template struct {
	ID          uuid.UUID          `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description,omitempty" db:"description"`
	Columns     []TemplateColumn   `json:"columns"`
	IsBuiltIn   bool               `json:"isBuiltIn" db:"is_built_in"`
	TeamID      *uuid.UUID         `json:"teamId,omitempty" db:"team_id"`
	CreatedBy   *uuid.UUID         `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt   time.Time          `json:"createdAt" db:"created_at"`
	PhaseTimes  map[RetroPhase]int `json:"phaseTimes,omitempty"`
}

// TemplateColumn represents a column in a template
type TemplateColumn struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"`
	Icon        string `json:"icon,omitempty"`
	Order       int    `json:"order"`
}

// Retrospective represents a retrospective session
type Retrospective struct {
	ID                    uuid.UUID          `json:"id" db:"id"`
	Name                  string             `json:"name" db:"name"`
	TeamID                uuid.UUID          `json:"teamId" db:"team_id"`
	TemplateID            uuid.UUID          `json:"templateId" db:"template_id"`
	FacilitatorID         uuid.UUID          `json:"facilitatorId" db:"facilitator_id"`
	Status                RetroStatus        `json:"status" db:"status"`
	CurrentPhase          RetroPhase         `json:"currentPhase" db:"current_phase"`
	MaxVotesPerUser       int                `json:"maxVotesPerUser" db:"max_votes_per_user"`
	MaxVotesPerItem       int                `json:"maxVotesPerItem" db:"max_votes_per_item"`
	AnonymousVoting       bool               `json:"anonymousVoting" db:"anonymous_voting"`
	AnonymousItems        bool               `json:"anonymousItems" db:"anonymous_items"`
	AllowItemEdit         bool               `json:"allowItemEdit" db:"allow_item_edit"`
	AllowVoteChange       bool               `json:"allowVoteChange" db:"allow_vote_change"`
	PhaseTimerOverrides   map[RetroPhase]int `json:"phaseTimerOverrides,omitempty" db:"phase_timer_overrides"`
	TimerStartedAt        *time.Time         `json:"timerStartedAt,omitempty" db:"timer_started_at"`
	TimerDurationSeconds  *int               `json:"timerDurationSeconds,omitempty" db:"timer_duration_seconds"`
	TimerPausedAt         *time.Time         `json:"timerPausedAt,omitempty" db:"timer_paused_at"`
	TimerRemainingSeconds *int               `json:"timerRemainingSeconds,omitempty" db:"timer_remaining_seconds"`
	ScheduledAt           *time.Time         `json:"scheduledAt,omitempty" db:"scheduled_at"`
	StartedAt             *time.Time         `json:"startedAt,omitempty" db:"started_at"`
	EndedAt               *time.Time         `json:"endedAt,omitempty" db:"ended_at"`
	RotiRevealed          bool               `json:"rotiRevealed" db:"roti_revealed"`
	CreatedAt             time.Time          `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time          `json:"updatedAt" db:"updated_at"`

	// Joined fields
	Team        *Team     `json:"team,omitempty"`
	Template    *Template `json:"template,omitempty"`
	Facilitator *User     `json:"facilitator,omitempty"`
}

// RetroParticipant represents a participant in a retrospective
type RetroParticipant struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	RetroID    uuid.UUID  `json:"retroId" db:"retro_id"`
	UserID     uuid.UUID  `json:"userId" db:"user_id"`
	IsOnline   bool       `json:"isOnline" db:"is_online"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty" db:"last_seen_at"`
	JoinedAt   time.Time  `json:"joinedAt" db:"joined_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// Item represents a card/item in a retrospective
type Item struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	RetroID   uuid.UUID  `json:"retroId" db:"retro_id"`
	ColumnID  string     `json:"columnId" db:"column_id"`
	Content   string     `json:"content" db:"content"`
	AuthorID  uuid.UUID  `json:"authorId" db:"author_id"`
	GroupID   *uuid.UUID `json:"groupId,omitempty" db:"group_id"`
	Position  int        `json:"position" db:"position"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`

	// Computed fields
	VoteCount int     `json:"voteCount"`
	Author    *User   `json:"author,omitempty"`
	Children  []*Item `json:"children,omitempty"`
}

// Vote represents a vote on an item
type Vote struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ItemID    uuid.UUID `json:"itemId" db:"item_id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ActionItem represents an action item from a retrospective
type ActionItem struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	RetroID     uuid.UUID  `json:"retroId" db:"retro_id"`
	ItemID      *uuid.UUID `json:"itemId,omitempty" db:"item_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	AssigneeID  *uuid.UUID `json:"assigneeId,omitempty" db:"assignee_id"`
	DueDate     *time.Time `json:"dueDate,omitempty" db:"due_date"`
	IsCompleted bool       `json:"isCompleted" db:"is_completed"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	Priority    int        `json:"priority" db:"priority"`
	ExternalID  *string    `json:"externalId,omitempty" db:"external_id"`
	ExternalURL *string    `json:"externalUrl,omitempty" db:"external_url"`
	CreatedBy   uuid.UUID  `json:"createdBy" db:"created_by"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`

	// Joined fields
	Assignee  *User  `json:"assignee,omitempty"`
	Item      *Item  `json:"item,omitempty"`
	RetroName string `json:"retroName,omitempty" db:"retro_name"`
}

// Integration represents an external integration
type Integration struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TeamID    uuid.UUID `json:"teamId" db:"team_id"`
	Type      string    `json:"type" db:"type"` // jira, slack, webhook
	Name      string    `json:"name" db:"name"`
	Config    string    `json:"-" db:"config"` // JSON encrypted
	IsEnabled bool      `json:"isEnabled" db:"is_enabled"`
	CreatedBy uuid.UUID `json:"createdBy" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// RecurringRetro represents a scheduled recurring retrospective
type RecurringRetro struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TeamID          uuid.UUID  `json:"teamId" db:"team_id"`
	TemplateID      uuid.UUID  `json:"templateId" db:"template_id"`
	Name            string     `json:"name" db:"name"`
	CronExpression  string     `json:"cronExpression" db:"cron_expression"`
	FacilitatorID   *uuid.UUID `json:"facilitatorId,omitempty" db:"facilitator_id"`
	IsEnabled       bool       `json:"isEnabled" db:"is_enabled"`
	NextScheduledAt *time.Time `json:"nextScheduledAt,omitempty" db:"next_scheduled_at"`
	LastRunAt       *time.Time `json:"lastRunAt,omitempty" db:"last_run_at"`
	CreatedBy       uuid.UUID  `json:"createdBy" db:"created_by"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
}

// TeamHealthSnapshot represents a health check snapshot
type TeamHealthSnapshot struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TeamID    uuid.UUID  `json:"teamId" db:"team_id"`
	RetroID   *uuid.UUID `json:"retroId,omitempty" db:"retro_id"`
	Period    string     `json:"period" db:"period"`
	Metrics   string     `json:"metrics" db:"metrics"` // JSON
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}

// RetroPhaseHistory represents phase timing history for analytics
type RetroPhaseHistory struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	RetroID                uuid.UUID  `json:"retroId" db:"retro_id"`
	Phase                  RetroPhase `json:"phase" db:"phase"`
	StartedAt              time.Time  `json:"startedAt" db:"started_at"`
	EndedAt                *time.Time `json:"endedAt,omitempty" db:"ended_at"`
	ActualDurationSeconds  *int       `json:"actualDurationSeconds,omitempty" db:"actual_duration_seconds"`
	PlannedDurationSeconds int        `json:"plannedDurationSeconds" db:"planned_duration_seconds"`
}

// IcebreakerMood represents a participant's mood in the icebreaker phase
type IcebreakerMood struct {
	ID        uuid.UUID   `json:"id" db:"id"`
	RetroID   uuid.UUID   `json:"retroId" db:"retro_id"`
	UserID    uuid.UUID   `json:"userId" db:"user_id"`
	Mood      MoodWeather `json:"mood" db:"mood"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
	User      *User       `json:"user,omitempty"`
}

// RotiVote represents a ROTI (Return On Time Invested) vote
type RotiVote struct {
	ID        uuid.UUID `json:"id" db:"id"`
	RetroID   uuid.UUID `json:"retroId" db:"retro_id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	Rating    int       `json:"rating" db:"rating"` // 1-5
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	User      *User     `json:"user,omitempty"`
}

// RotiResults represents aggregated ROTI results
type RotiResults struct {
	Average      float64     `json:"average"`
	TotalVotes   int         `json:"totalVotes"`
	Distribution map[int]int `json:"distribution"` // rating -> count
	Revealed     bool        `json:"revealed"`
	Votes        []*RotiVote `json:"votes,omitempty"`
}

// StatsFilter represents filter options for statistics queries
type StatsFilter struct {
	Limit     int        `json:"limit,omitempty"`
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
}

// RotiEvolutionPoint represents a ROTI data point in time
type RotiEvolutionPoint struct {
	RetroID   uuid.UUID `json:"retroId"`
	RetroName string    `json:"retroName"`
	Date      time.Time `json:"date"`
	Average   float64   `json:"average"`
	VoteCount int       `json:"voteCount"`
}

// MoodEvolutionPoint represents a mood data point in time
type MoodEvolutionPoint struct {
	RetroID      uuid.UUID           `json:"retroId"`
	RetroName    string              `json:"retroName"`
	Date         time.Time           `json:"date"`
	Distribution map[MoodWeather]int `json:"distribution"`
	MoodCount    int                 `json:"moodCount"`
}

// TeamRotiStats represents aggregated ROTI statistics for a team
type TeamRotiStats struct {
	Average           float64               `json:"average"`
	TotalVotes        int                   `json:"totalVotes"`
	TotalRetros       int                   `json:"totalRetros"`
	Distribution      map[int]int           `json:"distribution"` // rating -> count
	ParticipationRate float64               `json:"participationRate"`
	Evolution         []*RotiEvolutionPoint `json:"evolution"`
}

// TeamMoodStats represents aggregated mood statistics for a team
type TeamMoodStats struct {
	Distribution      map[MoodWeather]int   `json:"distribution"` // mood -> count
	TotalMoods        int                   `json:"totalMoods"`
	TotalRetros       int                   `json:"totalRetros"`
	ParticipationRate float64               `json:"participationRate"`
	Evolution         []*MoodEvolutionPoint `json:"evolution"`
}

// UserRotiStats represents ROTI statistics for a specific user
type UserRotiStats struct {
	UserID            uuid.UUID             `json:"userId"`
	Average           float64               `json:"average"`
	TotalVotes        int                   `json:"totalVotes"`
	RetrosAttended    int                   `json:"retrosAttended"`
	ParticipationRate float64               `json:"participationRate"`
	TeamAverage       float64               `json:"teamAverage"`
	Distribution      map[int]int           `json:"distribution"` // rating -> count
	Evolution         []*RotiEvolutionPoint `json:"evolution"`
}

// UserMoodStats represents mood statistics for a specific user
type UserMoodStats struct {
	UserID            uuid.UUID             `json:"userId"`
	Distribution      map[MoodWeather]int   `json:"distribution"` // mood -> count
	MostCommonMood    MoodWeather           `json:"mostCommonMood"`
	TotalMoods        int                   `json:"totalMoods"`
	RetrosAttended    int                   `json:"retrosAttended"`
	ParticipationRate float64               `json:"participationRate"`
	Evolution         []*MoodEvolutionPoint `json:"evolution"`
}

// CombinedUserStats represents combined ROTI and mood stats for a user
type CombinedUserStats struct {
	RotiStats *UserRotiStats `json:"rotiStats"`
	MoodStats *UserMoodStats `json:"moodStats"`
}

// RetroAttendee represents attendance record for a retrospective
type RetroAttendee struct {
	ID              uuid.UUID `json:"id" db:"id"`
	RetrospectiveID uuid.UUID `json:"retrospectiveId" db:"retrospective_id"`
	UserID          uuid.UUID `json:"userId" db:"user_id"`
	Attended        bool      `json:"attended" db:"attended"`
	RecordedAt      time.Time `json:"recordedAt" db:"recorded_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// TeamMemberStatus represents a team member with their connection status
type TeamMemberStatus struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Role        Role      `json:"role"`
	IsConnected bool      `json:"isConnected"`
}
