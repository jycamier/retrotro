export type Role = 'admin' | 'member'
export type SessionType = 'retro' | 'lean_coffee'
export type RetroPhase = 'waiting' | 'icebreaker' | 'brainstorm' | 'group' | 'vote' | 'discuss' | 'action' | 'roti' | 'propose'
export type RetroStatus = 'draft' | 'active' | 'completed' | 'archived'
export type MoodWeather = 'sunny' | 'partly_cloudy' | 'cloudy' | 'rainy' | 'stormy'

export interface User {
  id: string
  email: string
  displayName: string
  avatarUrl?: string
  isAdmin: boolean
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}

export interface Team {
  id: string
  name: string
  slug: string
  description?: string
  isOidcManaged: boolean
  createdBy?: string
  createdAt: string
  updatedAt: string
}

export interface TeamWithMemberCount extends Team {
  memberCount: number
}

export interface TeamMember {
  id: string
  teamId: string
  userId: string
  role: Role
  isOidcSynced: boolean
  lastSyncedAt?: string
  joinedAt: string
  user?: User
}

export interface TemplateColumn {
  id: string
  name: string
  description?: string
  color: string
  icon?: string
  order: number
}

export interface Template {
  id: string
  name: string
  description?: string
  columns: TemplateColumn[]
  isBuiltIn: boolean
  teamId?: string
  phaseTimes?: Record<RetroPhase, number>
  createdAt: string
}

export interface Retrospective {
  id: string
  name: string
  teamId: string
  templateId: string
  facilitatorId: string
  status: RetroStatus
  currentPhase: RetroPhase
  sessionType: SessionType
  maxVotesPerUser: number
  maxVotesPerItem: number
  anonymousVoting: boolean
  timerStartedAt?: string
  timerDurationSeconds?: number
  timerPausedAt?: string
  timerRemainingSeconds?: number
  scheduledAt?: string
  startedAt?: string
  endedAt?: string
  rotiRevealed: boolean
  lcCurrentTopicId?: string
  lcTopicTimeboxSeconds?: number
  createdAt: string
  updatedAt: string
  template?: Template
}

export interface Item {
  id: string
  retroId: string
  columnId: string
  content: string
  authorId: string
  groupId?: string
  position: number
  voteCount: number
  createdAt: string
  updatedAt: string
  author?: User
  children?: Item[]
}

export interface ActionItem {
  id: string
  retroId: string
  itemId?: string
  title: string
  description?: string
  assigneeId?: string
  dueDate?: string
  isCompleted: boolean
  status: 'todo' | 'in_progress' | 'done'
  completedAt?: string
  priority: number
  externalId?: string
  externalUrl?: string
  createdBy: string
  createdAt: string
  updatedAt: string
  assignee?: User
  itemContent?: string
  retroName?: string
}

export interface Participant {
  userId: string
  name: string
  voteCount?: number  // number of votes used (during vote phase)
}

// Team member with connection status (for waiting room)
export interface TeamMemberStatus {
  userId: string
  displayName: string
  avatarUrl?: string
  role: Role
  isConnected: boolean
}

// Draft item for anonymous typing (other users see masked content)
export interface DraftItem {
  userId: string
  userName: string
  columnId: string
  contentLength: number  // Length of the actual content (for generating masked version)
}

// WebSocket message types
export interface WSMessage<T = unknown> {
  type: string
  payload: T
}

export interface RetroState {
  retro: Retrospective
  items: Item[]
  actions: ActionItem[]
  participants: Participant[]
  timerRunning: boolean
  timerRemaining: number
}

// Icebreaker types
export interface IcebreakerMood {
  id: string
  retroId: string
  userId: string
  mood: MoodWeather
  createdAt: string
  user?: User
}

// ROTI types
export interface RotiVote {
  id: string
  retroId: string
  userId: string
  rating: number
  createdAt: string
  user?: User
}

export interface RotiResults {
  average: number
  totalVotes: number
  distribution: Record<number, number>
  revealed: boolean
  votes?: RotiVote[]
}

// API response types
export interface TokenPair {
  accessToken: string
  refreshToken: string
  expiresAt: string
}

export interface ApiError {
  error: string
  description?: string
}

// Statistics types
export interface StatsFilter {
  limit?: number
  startDate?: string
  endDate?: string
}

export interface RotiEvolutionPoint {
  retroId: string
  retroName: string
  date: string
  average: number
  voteCount: number
}

export interface MoodEvolutionPoint {
  retroId: string
  retroName: string
  date: string
  distribution: Record<MoodWeather, number>
  moodCount: number
}

export interface TeamRotiStats {
  average: number
  totalVotes: number
  totalRetros: number
  distribution: Record<number, number>
  participationRate: number
  evolution: RotiEvolutionPoint[]
}

export interface TeamMoodStats {
  distribution: Record<MoodWeather, number>
  totalMoods: number
  totalRetros: number
  participationRate: number
  evolution: MoodEvolutionPoint[]
}

export interface UserRotiStats {
  userId: string
  average: number
  totalVotes: number
  retrosAttended: number
  participationRate: number
  teamAverage: number
  distribution: Record<number, number>
  evolution: RotiEvolutionPoint[]
}

export interface UserMoodStats {
  userId: string
  distribution: Record<MoodWeather, number>
  mostCommonMood: MoodWeather
  totalMoods: number
  retrosAttended: number
  participationRate: number
  evolution: MoodEvolutionPoint[]
}

export interface CombinedUserStats {
  rotiStats: UserRotiStats
  moodStats: UserMoodStats
}

// Dev mode types
export interface DevUser {
  id: string
  email: string
  displayName: string
  isAdmin: boolean
  teamRole: Role
}

export interface DevTeam {
  id: string
  name: string
  slug: string
}

export interface DevUsersResponse {
  users: DevUser[]
  team: DevTeam
}

// Lean Coffee types
export interface LCDiscussionState {
  currentTopicId: string | null
  queue: Item[]
  done: Item[]
  topicHistory: LCTopicHistory[]
}

export interface LCTopicHistory {
  id: string
  retroId: string
  topicId: string
  discussionOrder: number
  totalDiscussionSeconds: number
  extensionCount: number
  startedAt: string
  endedAt?: string
}

export interface DiscussedTopic {
  id: string
  content: string
  authorId: string
  authorName: string
  sessionId: string
  sessionName: string
  discussedAt: string
  discussionOrder: number
  totalDiscussionSeconds: number
  extensionCount: number
}
