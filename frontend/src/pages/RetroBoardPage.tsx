import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { templatesApi } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import { useRetroStore } from '../store/retroStore'
import { useAuthStore } from '../store/authStore'
import RetroBoard from '../components/retrospective/RetroBoard'
import PhaseTimer from '../components/retrospective/PhaseTimer'
import ParticipantList from '../components/retrospective/ParticipantList'
import DiscussionCarousel from '../components/retrospective/DiscussionCarousel'
import IcebreakerPhaseView from '../components/retrospective/IcebreakerPhaseView'
import RotiPhaseView from '../components/retrospective/RotiPhaseView'
import RetroSummary from '../components/retrospective/RetroSummary'
import WaitingRoomView from '../components/retrospective/WaitingRoomView'
import { Loader2, LogOut, Users, Wifi, WifiOff } from 'lucide-react'

export default function RetroBoardPage() {
  const { retroId } = useParams<{ retroId: string }>()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const { retro, participants, items, actions, currentPhase, moods, rotiVotedUserIds, rotiResults, teamMembers, reset } = useRetroStore()
  const { isConnected, isStateLoaded, send, disconnect } = useWebSocket(retroId)
  const [showSummary, setShowSummary] = useState(false)

  // Reset store when retroId changes (entering a new retro)
  useEffect(() => {
    reset()
    setShowSummary(false)
  }, [retroId, reset])

  // Show summary when retro ends (only if it's the current retro)
  useEffect(() => {
    if (retro?.status === 'completed' && retro?.id === retroId) {
      setShowSummary(true)
    }
  }, [retro?.status, retro?.id, retroId])

  const { data: template } = useQuery({
    queryKey: ['template', retro?.templateId],
    queryFn: () => templatesApi.get(retro!.templateId),
    enabled: !!retro?.templateId,
  })

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect()
      reset()
    }
  }, [disconnect, reset])

  const isFacilitator = retro?.facilitatorId === user?.id

  const getAuthorName = (authorId: string): string => {
    const participant = participants.find(p => p.userId === authorId)
    if (participant) {
      return participant.name
    }
    if (participants.length === 0 && items.length > 0) {
      return '...'
    }
    if (authorId === user?.id && user?.displayName) {
      return user.displayName
    }
    return 'Inconnu'
  }

  const handleLeave = () => {
    disconnect()
    reset()
    navigate('/')
  }

  const phaseLabels: Record<string, string> = {
    waiting: 'Salle d\'attente',
    icebreaker: 'Icebreaker',
    brainstorm: 'Brainstorm',
    group: 'Group',
    vote: 'Vote',
    discuss: 'Discuss',
    roti: 'ROTI',
  }

  if (!retro || !template || !isStateLoaded) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-primary-600 animate-spin" />
      </div>
    )
  }

  // Check special phases for dedicated views
  const isWaitingPhase = currentPhase === 'waiting'
  const isIcebreakerPhase = currentPhase === 'icebreaker'
  const isDiscussPhase = currentPhase === 'discuss'
  const isRotiPhase = currentPhase === 'roti'

  return (
    <div className="min-h-screen bg-gray-100 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-gray-900">{retro.name}</h1>
            <span className="px-2 py-1 text-sm bg-primary-100 text-primary-700 rounded">
              {phaseLabels[currentPhase]}
            </span>
            {isConnected ? (
              <span className="flex items-center gap-1 text-green-600 text-sm">
                <Wifi className="w-4 h-4" />
                Connected
              </span>
            ) : (
              <span className="flex items-center gap-1 text-red-600 text-sm animate-pulse">
                <WifiOff className="w-4 h-4" />
                Disconnected
              </span>
            )}
          </div>

          <div className="flex items-center gap-4">
            <PhaseTimer isFacilitator={isFacilitator} send={send} />

            <div className="flex items-center gap-2">
              <Users className="w-4 h-4 text-gray-500" />
              <span className="text-sm text-gray-600">{participants.length}</span>
            </div>

            <button
              onClick={handleLeave}
              className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            >
              <LogOut className="w-4 h-4" />
              Leave
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {isWaitingPhase ? (
          <div className="flex-1 overflow-auto p-4">
            <WaitingRoomView
              teamMembers={teamMembers}
              facilitatorId={retro.facilitatorId}
              currentUserId={user?.id || ''}
              isFacilitator={isFacilitator}
              send={send}
            />
          </div>
        ) : isIcebreakerPhase ? (
          <div className="flex-1 overflow-auto p-4">
            <IcebreakerPhaseView
              moods={moods}
              participants={participants}
              currentUserId={user?.id || ''}
              isFacilitator={isFacilitator}
              send={send}
            />
          </div>
        ) : isDiscussPhase ? (
          <div className="flex-1">
            <DiscussionCarousel
              items={items}
              template={template}
              getAuthorName={getAuthorName}
              actions={actions}
              participants={participants}
              send={send}
              isFacilitator={isFacilitator}
            />
          </div>
        ) : isRotiPhase ? (
          <div className="flex-1 overflow-auto p-4">
            <RotiPhaseView
              rotiVotedUserIds={rotiVotedUserIds}
              rotiResults={rotiResults}
              participants={participants}
              currentUserId={user?.id || ''}
              isFacilitator={isFacilitator}
              send={send}
            />
          </div>
        ) : (
          <>
            <div className="flex-1 overflow-auto p-4">
              <RetroBoard
                template={template}
                currentPhase={currentPhase}
                isFacilitator={isFacilitator}
                send={send}
              />
            </div>

            <div className="w-44 bg-white border-l border-gray-200 p-3 flex flex-col">
              <ParticipantList
                participants={participants}
                facilitatorId={retro.facilitatorId}
                compact
              />

              {isFacilitator && (
                <div className="mt-4 pt-4 border-t border-gray-200">
                  <button
                    onClick={() => send('phase_next', {})}
                    className="w-full px-3 py-2 text-xs bg-primary-600 text-white rounded-lg hover:bg-primary-700"
                  >
                    Phase suivante
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Retro Summary Modal - shown when retro is completed */}
      {showSummary && template && retro && (
        <RetroSummary
          retro={retro}
          items={items}
          actions={actions}
          participants={participants}
          template={template}
          onClose={() => {
            setShowSummary(false)
            navigate('/')
          }}
        />
      )}
    </div>
  )
}
