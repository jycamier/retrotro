import { create } from 'zustand'
import type { Retrospective, Item, ActionItem, Participant, RetroPhase, MoodWeather, IcebreakerMood, RotiResults, TeamMemberStatus, DraftItem } from '../types'

interface RetroState {
  retro: Retrospective | null
  items: Item[]
  actions: ActionItem[]
  participants: Participant[]
  timerEndAt: Date | null
  isTimerRunning: boolean
  timerRemainingSeconds: number
  currentPhase: RetroPhase
  // Icebreaker state
  moods: Map<string, MoodWeather>  // userId -> mood
  // ROTI state
  rotiVotedUserIds: Set<string>
  rotiResults: RotiResults | null
  // Waiting room state
  teamMembers: TeamMemberStatus[]
  // Vote tracking (multi-vote)
  myVotesOnItems: Map<string, number>  // itemId -> number of my votes on that item

  // Draft items state (for anonymous typing during brainstorm)
  drafts: Map<string, DraftItem>  // key: "userId-columnId"

  // Synced discussion item (from discuss_item_changed)
  syncDiscussItemId: string | null

  // Actions
  setRetro: (retro: Retrospective) => void
  setItems: (items: Item[]) => void
  addItem: (item: Item) => void
  updateItem: (item: Item) => void
  removeItem: (itemId: string) => void
  setActions: (actions: ActionItem[]) => void
  addAction: (action: ActionItem) => void
  updateAction: (action: ActionItem) => void
  removeAction: (actionId: string) => void
  setParticipants: (participants: Participant[]) => void
  addParticipant: (participant: Participant) => void
  removeParticipant: (userId: string) => void

  // Timer
  setTimerStarted: (durationSeconds: number, endAt: string) => void
  setTimerPaused: (remainingSeconds: number) => void
  setTimerResumed: (remainingSeconds: number, endAt: string) => void
  setTimerEnded: () => void
  setTimerExtended: (newRemaining: number, newEndAt: string) => void
  updateTimerRemaining: (remaining: number) => void

  // Phase
  setPhase: (phase: RetroPhase) => void

  // Vote
  updateVote: (itemId: string, action: 'add' | 'remove', userId?: string, userVoteCount?: number) => void
  updateMyVoteOnItem: (itemId: string, action: 'add' | 'remove') => void
  setVoteSummary: (summary: Record<string, Record<string, number>>, currentUserId: string) => void

  // Grouping
  groupItems: (parentId: string, childIds: string[]) => void

  // Icebreaker
  setMoods: (moods: IcebreakerMood[]) => void
  updateMood: (userId: string, mood: MoodWeather) => void

  // ROTI
  setRotiVoteSubmitted: (userId: string) => void
  setRotiResults: (results: RotiResults) => void

  // Waiting room
  setTeamMembers: (members: TeamMemberStatus[]) => void
  updateTeamMemberStatus: (userId: string, isConnected: boolean) => void
  setFacilitator: (facilitatorId: string) => void

  // Drafts (anonymous typing)
  setDraft: (draft: DraftItem) => void
  clearDraft: (userId: string, columnId: string) => void

  // Discussion sync
  setSyncDiscussItemId: (itemId: string | null) => void

  reset: () => void
}

const initialState = {
  retro: null,
  items: [],
  actions: [],
  participants: [],
  timerEndAt: null,
  isTimerRunning: false,
  timerRemainingSeconds: 0,
  currentPhase: 'waiting' as RetroPhase,
  moods: new Map<string, MoodWeather>(),
  rotiVotedUserIds: new Set<string>(),
  rotiResults: null as RotiResults | null,
  teamMembers: [] as TeamMemberStatus[],
  myVotesOnItems: new Map<string, number>(),
  drafts: new Map<string, DraftItem>(),
  syncDiscussItemId: null as string | null,
}

export const useRetroStore = create<RetroState>((set) => ({
  ...initialState,

  setRetro: (retro) => set({ retro, currentPhase: retro.currentPhase }),

  setItems: (items) => set({ items }),

  addItem: (item) => set((state) => ({
    items: [...state.items, item],
  })),

  updateItem: (item) => set((state) => ({
    items: state.items.map((i) => i.id === item.id ? item : i),
  })),

  removeItem: (itemId) => set((state) => ({
    items: state.items.filter((i) => i.id !== itemId),
  })),

  setActions: (actions) => set({ actions }),

  addAction: (action) => set((state) => ({
    actions: [...state.actions, action],
  })),

  updateAction: (action) => set((state) => ({
    actions: state.actions.map((a) => a.id === action.id ? action : a),
  })),

  removeAction: (actionId) => set((state) => ({
    actions: state.actions.filter((a) => a.id !== actionId),
  })),

  setParticipants: (participants) => set({ participants }),

  addParticipant: (participant) => set((state) => ({
    participants: [...state.participants.filter(p => p.userId !== participant.userId), participant],
  })),

  removeParticipant: (userId) => set((state) => ({
    participants: state.participants.filter((p) => p.userId !== userId),
  })),

  setTimerStarted: (durationSeconds, endAt) => set({
    isTimerRunning: true,
    timerRemainingSeconds: durationSeconds,
    timerEndAt: new Date(endAt),
  }),

  setTimerPaused: (remainingSeconds) => set({
    isTimerRunning: false,
    timerRemainingSeconds: remainingSeconds,
    timerEndAt: null,
  }),

  setTimerResumed: (remainingSeconds, endAt) => set({
    isTimerRunning: true,
    timerRemainingSeconds: remainingSeconds,
    timerEndAt: new Date(endAt),
  }),

  setTimerEnded: () => set({
    isTimerRunning: false,
    timerRemainingSeconds: 0,
    timerEndAt: null,
  }),

  setTimerExtended: (newRemaining, newEndAt) => set({
    timerRemainingSeconds: newRemaining,
    timerEndAt: new Date(newEndAt),
  }),

  updateTimerRemaining: (remaining) => set({
    timerRemainingSeconds: remaining,
  }),

  setPhase: (phase) => set({ currentPhase: phase }),

  updateVote: (itemId, action, userId, userVoteCount) => set((state) => {
    // Update participant voteCount if userId and userVoteCount provided
    const newParticipants = (userId !== undefined && userVoteCount !== undefined)
      ? state.participants.map(p =>
          p.userId === userId ? { ...p, voteCount: userVoteCount } : p
        )
      : state.participants

    return {
      items: state.items.map((item) => {
        if (item.id === itemId) {
          return {
            ...item,
            voteCount: action === 'add' ? item.voteCount + 1 : Math.max(0, item.voteCount - 1),
          }
        }
        return item
      }),
      participants: newParticipants,
    }
  }),

  // Update myVotesOnItems when the current user votes
  updateMyVoteOnItem: (itemId: string, action: 'add' | 'remove') => set((state) => {
    const newMyVotes = new Map(state.myVotesOnItems)
    const current = newMyVotes.get(itemId) || 0
    if (action === 'add') {
      newMyVotes.set(itemId, current + 1)
    } else {
      newMyVotes.set(itemId, Math.max(0, current - 1))
    }
    return { myVotesOnItems: newMyVotes }
  }),

  setVoteSummary: (summary, currentUserId) => set((state) => {
    // Initialize myVotesOnItems from the summary for the current user
    const newMyVotes = new Map<string, number>()
    const myVotes = summary[currentUserId]
    if (myVotes) {
      for (const [itemId, count] of Object.entries(myVotes)) {
        newMyVotes.set(itemId, count)
      }
    }

    // Compute voteCount per participant (total votes used)
    const newParticipants = state.participants.map(p => {
      const userVotes = summary[p.userId]
      if (userVotes) {
        const totalVotes = Object.values(userVotes).reduce((sum, count) => sum + count, 0)
        return { ...p, voteCount: totalVotes }
      }
      return { ...p, voteCount: 0 }
    })

    return {
      myVotesOnItems: newMyVotes,
      participants: newParticipants,
    }
  }),

  groupItems: (parentId, childIds) => set((state) => ({
    items: state.items.map((item) => {
      if (childIds.includes(item.id)) {
        return { ...item, groupId: parentId }
      }
      return item
    }),
  })),

  // Icebreaker
  setMoods: (moods) => set(() => {
    const moodMap = new Map<string, MoodWeather>()
    moods.forEach((m) => moodMap.set(m.userId, m.mood))
    return { moods: moodMap }
  }),

  updateMood: (userId, mood) => set((state) => {
    const newMoods = new Map(state.moods)
    newMoods.set(userId, mood)
    return { moods: newMoods }
  }),

  // ROTI
  setRotiVoteSubmitted: (userId) => set((state) => {
    const newVotedIds = new Set(state.rotiVotedUserIds)
    newVotedIds.add(userId)
    return { rotiVotedUserIds: newVotedIds }
  }),

  setRotiResults: (results) => set({ rotiResults: results }),

  // Waiting room
  setTeamMembers: (members) => set({ teamMembers: members }),

  updateTeamMemberStatus: (userId, isConnected) => set((state) => ({
    teamMembers: state.teamMembers.map((member) =>
      member.userId === userId ? { ...member, isConnected } : member
    ),
  })),

  setFacilitator: (facilitatorId) => set((state) => ({
    retro: state.retro ? { ...state.retro, facilitatorId } : null,
  })),

  // Drafts (anonymous typing)
  setDraft: (draft) => set((state) => {
    const newDrafts = new Map(state.drafts)
    const key = `${draft.userId}-${draft.columnId}`
    newDrafts.set(key, draft)
    return { drafts: newDrafts }
  }),

  clearDraft: (userId, columnId) => set((state) => {
    const newDrafts = new Map(state.drafts)
    const key = `${userId}-${columnId}`
    newDrafts.delete(key)
    return { drafts: newDrafts }
  }),

  reset: () => set({
    ...initialState,
    moods: new Map<string, MoodWeather>(),
    rotiVotedUserIds: new Set<string>(),
    rotiResults: null,
    teamMembers: [],
    myVotesOnItems: new Map<string, number>(),
    drafts: new Map<string, DraftItem>(),
  }),
}))
