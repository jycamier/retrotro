import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useWebSocket } from '../hooks/useWebSocket'
import { useRetroStore } from '../store/retroStore'
import { useLeanCoffeeStore } from '../store/leanCoffeeStore'
import { useAuthStore } from '../store/authStore'
import PhaseTimer from '../components/retrospective/PhaseTimer'
import ParticipantList from '../components/retrospective/ParticipantList'
import IcebreakerPhaseView from '../components/retrospective/IcebreakerPhaseView'
import RotiPhaseView from '../components/retrospective/RotiPhaseView'
import RetroSummary from '../components/retrospective/RetroSummary'
import WaitingRoomView from '../components/retrospective/WaitingRoomView'
import LCProposePhaseView from '../components/leancoffee/LCProposePhaseView'
import LCVotePhaseView from '../components/leancoffee/LCVotePhaseView'
import LCDiscussPhaseView from '../components/leancoffee/LCDiscussPhaseView'
import { Loader2, LogOut, Users, Wifi, WifiOff, Coffee } from 'lucide-react'

export default function LeanCoffeeBoardPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const { retro, participants, items, actions, currentPhase, moods, rotiVotedUserIds, rotiResults, teamMembers, reset } = useRetroStore()
  const lcReset = useLeanCoffeeStore(s => s.reset)
  const { isConnected, isStateLoaded, send, disconnect } = useWebSocket(sessionId)
  const [showSummary, setShowSummary] = useState(false)

  // Reset stores when sessionId changes
  useEffect(() => {
    reset()
    lcReset()
    setShowSummary(false)
  }, [sessionId, reset, lcReset])

  // Show summary when session ends
  useEffect(() => {
    if (retro?.status === 'completed' && retro?.id === sessionId) {
      setShowSummary(true)
    }
  }, [retro?.status, retro?.id, sessionId])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect()
      reset()
      lcReset()
    }
  }, [disconnect, reset, lcReset])

  const isFacilitator = retro?.facilitatorId === user?.id

  const handleLeave = () => {
    disconnect()
    reset()
    lcReset()
    navigate('/')
  }

  const phaseLabels: Record<string, string> = {
    waiting: 'Salle d\'attente',
    icebreaker: 'Icebreaker',
    propose: 'Propositions',
    vote: 'Vote',
    discuss: 'Discussion',
    roti: 'ROTI',
  }

  if (!retro || !isStateLoaded) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-primary-600 animate-spin" />
      </div>
    )
  }

  const isWaitingPhase = currentPhase === 'waiting'
  const isIcebreakerPhase = currentPhase === 'icebreaker'
  const isProposePhase = currentPhase === 'propose'
  const isVotePhase = currentPhase === 'vote'
  const isDiscussPhase = currentPhase === 'discuss'
  const isRotiPhase = currentPhase === 'roti'

  // Phases with sidebar (propose and vote)
  const showSidebar = isProposePhase || isVotePhase

  return (
    <div className="min-h-screen bg-gray-100 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Coffee className="w-5 h-5 text-amber-600" />
              <h1 className="text-lg font-semibold text-gray-900">{retro.name}</h1>
            </div>
            <span className="px-2 py-1 text-sm bg-amber-100 text-amber-700 rounded">
              {phaseLabels[currentPhase] || currentPhase}
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
            {!isDiscussPhase && (
              <PhaseTimer isFacilitator={isFacilitator} send={send} />
            )}

            <div className="flex items-center gap-2">
              <Users className="w-4 h-4 text-gray-500" />
              <span className="text-sm text-gray-600">{participants.length}</span>
            </div>

            <button
              onClick={handleLeave}
              className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            >
              <LogOut className="w-4 h-4" />
              Quitter
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
        ) : isProposePhase ? (
          <>
            <div className="flex-1 overflow-auto p-4">
              <LCProposePhaseView send={send} isFacilitator={isFacilitator} />
            </div>
            {showSidebar && (
              <div className="w-44 bg-white border-l border-gray-200 p-3 flex flex-col">
                <ParticipantList
                  participants={participants}
                  facilitatorId={retro.facilitatorId}
                  compact
                  currentPhase={currentPhase}
                  maxVotesPerUser={retro.maxVotesPerUser}
                />
              </div>
            )}
          </>
        ) : isVotePhase ? (
          <>
            <div className="flex-1 overflow-auto p-4">
              <LCVotePhaseView send={send} isFacilitator={isFacilitator} />
            </div>
            {showSidebar && (
              <div className="w-44 bg-white border-l border-gray-200 p-3 flex flex-col">
                <ParticipantList
                  participants={participants}
                  facilitatorId={retro.facilitatorId}
                  compact
                  currentPhase={currentPhase}
                  maxVotesPerUser={retro.maxVotesPerUser}
                />
              </div>
            )}
          </>
        ) : isDiscussPhase ? (
          <div className="flex-1">
            <LCDiscussPhaseView
              send={send}
              isFacilitator={isFacilitator}
              actions={actions}
              participants={participants}
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
          <div className="flex-1 flex items-center justify-center text-gray-500">
            Phase inconnue : {currentPhase}
          </div>
        )}
      </div>

      {/* Summary Modal */}
      {showSummary && retro && (
        <RetroSummary
          retro={retro}
          items={items}
          actions={actions}
          participants={participants}
          template={retro.template || { id: '', name: 'Lean Coffee', columns: [{ id: 'topics', name: 'Topics', color: '#f59e0b', order: 0 }], isBuiltIn: true, createdAt: '' }}
          onClose={() => {
            setShowSummary(false)
            navigate('/')
          }}
        />
      )}
    </div>
  )
}
