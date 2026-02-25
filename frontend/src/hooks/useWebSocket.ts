import { useEffect, useRef, useCallback, useState } from 'react'
import { useAuthStore } from '../store/authStore'
import { useRetroStore } from '../store/retroStore'
import { useLeanCoffeeStore } from '../store/leanCoffeeStore'
import type { WSMessage, Item, RetroPhase, IcebreakerMood, RotiResults, MoodWeather, TeamMemberStatus, DraftItem, Participant, LCDiscussionState } from '../types'

interface ExtendedRetroState {
  retro: import('../types').Retrospective
  items: Item[]
  actions: import('../types').ActionItem[]
  participants: import('../types').Participant[]
  timerRunning: boolean
  timerRemaining: number
  moods: IcebreakerMood[]
  rotiResults: RotiResults | null
  teamMembers: TeamMemberStatus[] | null
  voteSummary: Record<string, Record<string, number>> | null
  lcDiscussionState?: LCDiscussionState | null
}

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`

// Session storage keys for backup state during reload
const PARTICIPANTS_BACKUP_KEY = 'retro-participants-backup'
const STATE_RECEIVED_KEY = 'retro-state-received'

// Helper to save participants to sessionStorage as backup
function saveParticipantsBackup(retroId: string, participants: Participant[]) {
  try {
    sessionStorage.setItem(`${PARTICIPANTS_BACKUP_KEY}-${retroId}`, JSON.stringify(participants))
  } catch {
    // Ignore storage errors
  }
}

// Helper to restore participants from sessionStorage backup
function restoreParticipantsBackup(retroId: string): Participant[] | null {
  try {
    const backup = sessionStorage.getItem(`${PARTICIPANTS_BACKUP_KEY}-${retroId}`)
    if (backup) {
      return JSON.parse(backup)
    }
  } catch {
    // Ignore parse errors
  }
  return null
}

// Helper to clear participants backup
function clearParticipantsBackup(retroId: string) {
  try {
    sessionStorage.removeItem(`${PARTICIPANTS_BACKUP_KEY}-${retroId}`)
    sessionStorage.removeItem(`${STATE_RECEIVED_KEY}-${retroId}`)
  } catch {
    // Ignore storage errors
  }
}

export function useWebSocket(retroId: string | undefined) {
  const { accessToken } = useAuthStore()
  const retroStore = useRetroStore()
  const wsRef = useRef<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [isStateLoaded, setIsStateLoaded] = useState(false)
  const isStateLoadedRef = useRef(false) // Ref for use in timeouts (avoids closure issues)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>()
  const stateTimeoutRef = useRef<ReturnType<typeof setTimeout>>()
  const heartbeatTimeoutRef = useRef<ReturnType<typeof setInterval>>() // Heartbeat interval
  const reconnectAttempts = useRef(0)
  const joinRetryAttempts = useRef(0)
  const maxReconnectAttempts = 5
  const maxJoinRetryAttempts = 3
  const intentionalDisconnectRef = useRef(false)
  const heartbeatIntervalMs = 30_000 // Send heartbeat every 30 seconds

  const connect = useCallback(() => {
    if (!retroId || !accessToken) return

    console.log('[WS] connect() called, reconnectAttempts:', reconnectAttempts.current)

    // Restore participants backup while waiting for retro_state
    const backupParticipants = restoreParticipantsBackup(retroId)
    if (backupParticipants && backupParticipants.length > 0) {
      console.log('[WS] Restoring participants from backup:', backupParticipants.length)
      retroStore.setParticipants(backupParticipants)
    }

    const ws = new WebSocket(`${WS_URL}?token=${accessToken}`)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      setConnectionError(null)
      reconnectAttempts.current = 0
      joinRetryAttempts.current = 0

      // Join retro room
      send('join_retro', { retroId })

      // Start heartbeat to keep connection alive (especially important with latency)
      if (heartbeatTimeoutRef.current) {
        clearInterval(heartbeatTimeoutRef.current)
      }
      heartbeatTimeoutRef.current = setInterval(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          console.log('[WS] Sending heartbeat to keep connection alive')
          send('heartbeat', {})
        }
      }, heartbeatIntervalMs)

      // Set timeout to retry join if retro_state doesn't arrive
      if (stateTimeoutRef.current) {
        clearTimeout(stateTimeoutRef.current)
      }
      stateTimeoutRef.current = setTimeout(() => {
        if (!isStateLoadedRef.current && joinRetryAttempts.current < maxJoinRetryAttempts) {
          console.log('[WS] retro_state not received, retrying join_retro, attempt:', joinRetryAttempts.current + 1)
          joinRetryAttempts.current++
          send('join_retro', { retroId })

          // Set another timeout for next retry
          stateTimeoutRef.current = setTimeout(() => {
            if (!isStateLoadedRef.current) {
              console.error('[WS] Failed to receive retro_state after retries')
              setConnectionError('Impossible de charger l\'état de la rétrospective. Veuillez rafraîchir la page.')
            }
          }, 5000)
        }
      }, 3000)
    }

    ws.onclose = () => {
      console.log('[WS] onclose triggered, intentionalDisconnect:', intentionalDisconnectRef.current)
      setIsConnected(false)
      wsRef.current = null

      // Don't reconnect if disconnect was intentional (user clicked Leave)
      if (intentionalDisconnectRef.current) {
        console.log('[WS] intentional disconnect, skipping reconnect')
        intentionalDisconnectRef.current = false
        return
      }

      // Attempt reconnection only for unexpected disconnects
      if (reconnectAttempts.current < maxReconnectAttempts) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000)
        console.log('[WS] scheduling reconnect in', delay, 'ms, attempt:', reconnectAttempts.current + 1)
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectAttempts.current++
          connect()
        }, delay)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onmessage = (event) => {
      // Backend may batch multiple messages separated by newlines
      const messages = event.data.split('\n').filter((line: string) => line.trim())
      for (const msgStr of messages) {
        try {
          const message: WSMessage = JSON.parse(msgStr)
          handleMessage(message)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error, msgStr)
        }
      }
    }
  }, [retroId, accessToken])

  const handleMessage = useCallback((message: WSMessage) => {
    const { type, payload } = message

    switch (type) {
      case 'retro_state': {
        const state = payload as ExtendedRetroState
        console.log('[WS] retro_state received, participants:', state.participants?.length)

        // Clear the state timeout since we received the state
        if (stateTimeoutRef.current) {
          clearTimeout(stateTimeoutRef.current)
        }
        setIsStateLoaded(true)
        isStateLoadedRef.current = true
        setConnectionError(null)

        retroStore.setRetro(state.retro)
        retroStore.setItems(state.items || [])
        retroStore.setActions(state.actions || [])
        retroStore.setParticipants(state.participants || [])

        // Save participants backup for reload recovery
        if (retroId && state.participants) {
          saveParticipantsBackup(retroId, state.participants)
        }

        if (state.timerRunning && state.timerRemaining > 0) {
          const endAt = new Date(Date.now() + state.timerRemaining * 1000)
          retroStore.setTimerStarted(state.timerRemaining, endAt.toISOString())
        }
        // Set icebreaker moods
        if (state.moods) {
          retroStore.setMoods(state.moods)
        }
        // Set ROTI results
        if (state.rotiResults) {
          retroStore.setRotiResults(state.rotiResults)
        }
        // Set team members (for waiting room)
        if (state.teamMembers) {
          retroStore.setTeamMembers(state.teamMembers)
        }
        // Set vote summary (for multi-vote tracking)
        if (state.voteSummary) {
          const currentUserId = useAuthStore.getState().user?.id || ''
          retroStore.setVoteSummary(state.voteSummary, currentUserId)
        }
        // Set LC discussion state if present
        if (state.lcDiscussionState) {
          useLeanCoffeeStore.getState().setDiscussionState(state.lcDiscussionState)
        }
        break
      }

      case 'error': {
        const { code, message: errorMessage } = payload as { code: string; message: string }
        console.error('[WS] Server error:', code, errorMessage)
        setConnectionError(errorMessage)
        break
      }

      case 'item_created':
        console.log('[WS] item_created received:', payload)
        retroStore.addItem(payload as Item)
        break

      case 'item_updated':
        retroStore.updateItem(payload as Item)
        break

      case 'item_deleted': {
        const { itemId } = payload as { itemId: string }
        retroStore.removeItem(itemId)
        break
      }

      case 'vote_updated': {
        const { itemId, action, userId, userVoteCount } = payload as { itemId: string; action: 'add' | 'remove'; userId: string; userVoteCount: number }
        retroStore.updateVote(itemId, action, userId, userVoteCount)
        // Track personal votes
        const currentUserId = useAuthStore.getState().user?.id
        if (userId === currentUserId) {
          retroStore.updateMyVoteOnItem(itemId, action)
        }
        break
      }

      case 'participant_joined': {
        const { userId, name } = payload as { userId: string; name: string }
        retroStore.addParticipant({ userId, name })
        // Update backup (use store directly to get current state)
        if (retroId) {
          const currentParticipants = useRetroStore.getState().participants
          saveParticipantsBackup(retroId, currentParticipants)
        }
        break
      }

      case 'participant_left': {
        const { userId } = payload as { userId: string }
        retroStore.removeParticipant(userId)
        // Also update team member status for waiting room
        retroStore.updateTeamMemberStatus(userId, false)
        // Update backup (use store directly to get current state)
        if (retroId) {
          const currentParticipants = useRetroStore.getState().participants
          saveParticipantsBackup(retroId, currentParticipants)
        }
        break
      }

      case 'timer_started': {
        const { duration_seconds, end_at } = payload as { duration_seconds: number; end_at: string }
        retroStore.setTimerStarted(duration_seconds, end_at)
        break
      }

      case 'timer_tick': {
        const { remaining_seconds } = payload as { remaining_seconds: number }
        retroStore.updateTimerRemaining(remaining_seconds)
        break
      }

      case 'timer_paused': {
        const { remaining_seconds } = payload as { remaining_seconds: number }
        retroStore.setTimerPaused(remaining_seconds)
        break
      }

      case 'timer_resumed': {
        const { remaining_seconds, end_at } = payload as { remaining_seconds: number; end_at: string }
        retroStore.setTimerResumed(remaining_seconds, end_at)
        break
      }

      case 'timer_ended':
        retroStore.setTimerEnded()
        break

      case 'timer_extended': {
        const { new_remaining, new_end_at } = payload as { new_remaining: number; new_end_at: string }
        retroStore.setTimerExtended(new_remaining, new_end_at)
        break
      }

      case 'phase_changed': {
        const { current_phase } = payload as { current_phase: RetroPhase }
        retroStore.setPhase(current_phase)
        break
      }

      case 'items_grouped': {
        const { parentId, childIds } = payload as { parentId: string; childIds: string[] }
        retroStore.groupItems(parentId, childIds)
        break
      }

      case 'action_created':
        retroStore.addAction(payload as import('../types').ActionItem)
        break

      case 'action_updated':
        retroStore.updateAction(payload as import('../types').ActionItem)
        break

      case 'action_deleted': {
        const { actionId } = payload as { actionId: string }
        retroStore.removeAction(actionId)
        break
      }

      case 'retro_ended': {
        const { retro, rotiResults } = payload as { retro: import('../types').Retrospective; rotiResults?: RotiResults }
        retroStore.setRetro(retro)
        if (rotiResults) {
          retroStore.setRotiResults(rotiResults)
        }
        break
      }

      case 'mood_updated': {
        const { userId, mood } = payload as { userId: string; mood: MoodWeather }
        retroStore.updateMood(userId, mood)
        break
      }

      case 'roti_vote_submitted': {
        const { userId } = payload as { userId: string }
        retroStore.setRotiVoteSubmitted(userId)
        break
      }

      case 'roti_results_revealed': {
        retroStore.setRotiResults(payload as RotiResults)
        break
      }

      case 'team_members_updated': {
        const { teamMembers } = payload as { teamMembers: TeamMemberStatus[] }
        console.log('[WS] team_members_updated received:', teamMembers)
        retroStore.setTeamMembers(teamMembers)
        break
      }

      case 'facilitator_changed': {
        const { facilitatorId } = payload as { facilitatorId: string; facilitatorName: string }
        console.log('[WS] facilitator_changed received:', facilitatorId)
        retroStore.setFacilitator(facilitatorId)
        break
      }

      case 'draft_typing': {
        const draft = payload as DraftItem
        retroStore.setDraft(draft)
        break
      }

      case 'draft_cleared': {
        const { userId, columnId } = payload as { userId: string; columnId: string }
        retroStore.clearDraft(userId, columnId)
        break
      }
    }
  }, [retroStore])

  const send = useCallback((type: string, payload: Record<string, unknown>) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      // If sending leave_retro, mark as intentional disconnect to prevent auto-reconnect
      if (type === 'leave_retro') {
        intentionalDisconnectRef.current = true
      }
      console.log('[WS] Sending message:', type, payload)
      wsRef.current.send(JSON.stringify({ type, payload }))
    } else {
      console.warn('[WS] Cannot send message, WebSocket not ready:', type, wsRef.current?.readyState)
    }
  }, [])

  const disconnect = useCallback(() => {
    intentionalDisconnectRef.current = true
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (stateTimeoutRef.current) {
      clearTimeout(stateTimeoutRef.current)
    }
    if (heartbeatTimeoutRef.current) {
      clearInterval(heartbeatTimeoutRef.current)
    }
    if (wsRef.current) {
      send('leave_retro', {})
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
    setIsStateLoaded(false)
    isStateLoadedRef.current = false
    // Clear backup when user intentionally leaves
    if (retroId) {
      clearParticipantsBackup(retroId)
    }
  }, [send, retroId])

  useEffect(() => {
    connect()

    // Handle page unload (tab close, browser close, navigation away)
    const handleBeforeUnload = () => {
      console.log('[WS] beforeunload triggered, marking disconnect as intentional')
      intentionalDisconnectRef.current = true
      // Note: We keep the backup on beforeunload for page reload recovery
    }
    window.addEventListener('beforeunload', handleBeforeUnload)

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
      if (stateTimeoutRef.current) {
        clearTimeout(stateTimeoutRef.current)
      }
      if (heartbeatTimeoutRef.current) {
        clearInterval(heartbeatTimeoutRef.current)
      }
      disconnect()
    }
  }, [connect, disconnect])

  return {
    isConnected,
    isStateLoaded,
    connectionError,
    send,
    disconnect,
  }
}
