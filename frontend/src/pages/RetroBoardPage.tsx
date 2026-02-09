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
import ActionPhaseView from '../components/retrospective/ActionPhaseView'
import IcebreakerPhaseView from '../components/retrospective/IcebreakerPhaseView'
import RotiPhaseView from '../components/retrospective/RotiPhaseView'
import RetroSummary from '../components/retrospective/RetroSummary'
import WaitingRoomView from '../components/retrospective/WaitingRoomView'
import { LogOut, Users, Wifi, WifiOff, MessageSquare, CheckCircle, Loader2 } from 'lucide-react'

export default function RetroBoardPage() {
  const { retroId } = useParams<{ retroId: string }>()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const { retro, participants, items, actions, currentPhase, moods, rotiVotedUserIds, rotiResults, teamMembers, reset } = useRetroStore()
  const { isConnected, isStateLoaded, connectionError, send, disconnect } = useWebSocket(retroId)
  const [isDiscussionOpen, setIsDiscussionOpen] = useState(false)
  const [showSummary, setShowSummary] = useState(false)

  // Reset store when retroId changes (entering a new retro)
  useEffect(() => {
    // Reset state when entering a new retro
    reset()
    setShowSummary(false)
    setIsDiscussionOpen(false)
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
    // If no participants but we have items, we're likely in a loading state (page reload)
    if (participants.length === 0 && items.length > 0) {
      return '...'
    }
    // Check if the author is the current user
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
    action: 'Actions',
    roti: 'ROTI',
  }

  if (!isConnected || !isStateLoaded || !retro || !template) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          {connectionError ? (
            <>
              <div className="text-red-500 mb-4">
                <WifiOff className="w-12 h-12 mx-auto" />
              </div>
              <p className="text-red-600 font-medium">{connectionError}</p>
              <button
                onClick={() => window.location.reload()}
                className="mt-4 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                Rafraîchir la page
              </button>
            </>
          ) : (
            <div className="space-y-4">
              {/* Step 1 — Connexion */}
              <div className="flex items-center gap-3">
                {isConnected ? (
                  <CheckCircle className="w-5 h-5 text-green-500" />
                ) : (
                  <Loader2 className="w-5 h-5 text-primary-600 animate-spin" />
                )}
                <span className={isConnected ? 'text-gray-400' : 'text-gray-700 font-medium'}>
                  Connexion au serveur…
                </span>
              </div>

              {/* Step 2 — Chargement des données */}
              <div className="flex items-center gap-3">
                {!isConnected ? (
                  <div className="w-5 h-5 rounded-full border-2 border-gray-300" />
                ) : isStateLoaded ? (
                  <CheckCircle className="w-5 h-5 text-green-500" />
                ) : (
                  <Loader2 className="w-5 h-5 text-primary-600 animate-spin" />
                )}
                <span className={!isConnected ? 'text-gray-300' : isStateLoaded ? 'text-gray-400' : 'text-gray-700 font-medium'}>
                  Chargement des données…
                </span>
              </div>

              {/* Step 3 — Préparation */}
              <div className="flex items-center gap-3">
                {!isStateLoaded ? (
                  <div className="w-5 h-5 rounded-full border-2 border-gray-300" />
                ) : retro && template ? (
                  <CheckCircle className="w-5 h-5 text-green-500" />
                ) : (
                  <Loader2 className="w-5 h-5 text-primary-600 animate-spin" />
                )}
                <span className={!isStateLoaded ? 'text-gray-300' : retro && template ? 'text-gray-400' : 'text-gray-700 font-medium'}>
                  Préparation de la rétrospective…
                </span>
              </div>
            </div>
          )}
        </div>
      </div>
    )
  }

  // Check special phases for dedicated views
  const isWaitingPhase = currentPhase === 'waiting'
  const isIcebreakerPhase = currentPhase === 'icebreaker'
  const isActionPhase = currentPhase === 'action'
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
            {(currentPhase === 'discuss' || currentPhase === 'vote') && (
              <button
                onClick={() => setIsDiscussionOpen(true)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                <MessageSquare className="w-4 h-4" />
                Ouvrir la discussion
              </button>
            )}
            {isConnected ? (
              <span className="flex items-center gap-1 text-green-600 text-sm">
                <Wifi className="w-4 h-4" />
                Connected
              </span>
            ) : (
              <span className="flex items-center gap-1 text-red-600 text-sm">
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
          /* Waiting Room Phase - Full width waiting room view */
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
          /* Icebreaker Phase - Full width mood selection view */
          <div className="flex-1 overflow-auto p-4">
            <IcebreakerPhaseView
              moods={moods}
              participants={participants}
              currentUserId={user?.id || ''}
              isFacilitator={isFacilitator}
              send={send}
            />
          </div>
        ) : isRotiPhase ? (
          /* ROTI Phase - Full width rating view */
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
        ) : isActionPhase ? (
          /* Action Phase - Dedicated 2-column view */
          <div className="flex-1 overflow-auto p-4">
            <ActionPhaseView
              items={items}
              actions={actions}
              participants={participants}
              template={template}
              isFacilitator={isFacilitator}
              send={send}
            />
          </div>
        ) : (
          /* Other phases - Normal board view */
          <>
            <div className="flex-1 overflow-auto p-4">
              <RetroBoard
                template={template}
                currentPhase={currentPhase}
                isFacilitator={isFacilitator}
                send={send}
              />
            </div>

            {/* Sidebar - compact view */}
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

      {/* Discussion Carousel Modal */}
      {template && (
        <DiscussionCarousel
          items={items}
          template={template}
          isOpen={isDiscussionOpen}
          onClose={() => setIsDiscussionOpen(false)}
          getAuthorName={getAuthorName}
        />
      )}

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
