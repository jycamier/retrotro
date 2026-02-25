import { useState, useMemo } from 'react'
import { useLeanCoffeeStore } from '../../store/leanCoffeeStore'
import { useRetroStore } from '../../store/retroStore'
import { useAuthStore } from '../../store/authStore'
import PhaseTimer from '../retrospective/PhaseTimer'
import { SkipForward, Plus, Check, Trash2, User, Calendar, ThumbsUp, MessageSquare, Clock, CheckCircle2, ListChecks, ArrowRight } from 'lucide-react'
import clsx from 'clsx'
import type { ActionItem, Item, Participant } from '../../types'

interface LCDiscussPhaseViewProps {
  send: (type: string, payload: Record<string, unknown>) => void
  isFacilitator: boolean
  actions: ActionItem[]
  participants: Participant[]
}

export default function LCDiscussPhaseView({ send, isFacilitator, actions, participants }: LCDiscussPhaseViewProps) {
  const { currentTopicId, queue, done, topicHistory, allTopicsDone } = useLeanCoffeeStore()
  const { items, retro } = useRetroStore()
  const { user } = useAuthStore()
  const [newActionTitle, setNewActionTitle] = useState('')
  const [newActionAssignee, setNewActionAssignee] = useState('')

  const currentTopic = useMemo(() => {
    if (!currentTopicId) return null
    return items.find(i => i.id === currentTopicId) || null
  }, [currentTopicId, items])

  const currentTopicActions = useMemo(() => {
    if (!currentTopicId) return []
    return actions.filter(a => a.itemId === currentTopicId)
  }, [currentTopicId, actions])

  const handleNextTopic = () => {
    // Pick the first item from the queue
    if (queue.length > 0) {
      send('discuss_set_item', { itemId: queue[0].id })
    }
  }

  const handleSelectTopic = (topicId: string) => {
    if (!isFacilitator) return
    send('discuss_set_item', { itemId: topicId })
  }

  const handleAddTime = () => {
    send('timer_add_time', { seconds: 300 }) // +5 min
  }

  const handleCreateAction = () => {
    if (!newActionTitle.trim()) return
    send('action_create', {
      title: newActionTitle.trim(),
      assigneeId: newActionAssignee || undefined,
      itemId: currentTopicId || undefined,
    })
    setNewActionTitle('')
    setNewActionAssignee('')
  }

  const handleToggleComplete = (action: ActionItem) => {
    if (action.isCompleted) {
      send('action_uncomplete', { actionId: action.id })
    } else {
      send('action_complete', { actionId: action.id })
    }
  }

  const handleDeleteAction = (actionId: string) => {
    send('action_delete', { actionId })
  }

  const getHistoryForTopic = (topicId: string) => {
    return topicHistory.find(h => h.topicId === topicId)
  }

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    if (mins > 0) return `${mins}m${secs > 0 ? ` ${secs}s` : ''}`
    return `${secs}s`
  }

  if (allTopicsDone && !currentTopic) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-gray-500">
        <CheckCircle2 className="w-12 h-12 text-green-500 mb-4" />
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Tous les sujets ont été discutés !</h2>
        <p className="text-sm text-gray-500 mb-6">{done.length} sujets discutés</p>
        {isFacilitator && (
          <button
            onClick={() => send('phase_next', {})}
            className="flex items-center gap-2 px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm font-medium"
          >
            Passer au ROTI
            <ArrowRight className="w-4 h-4" />
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with timer and facilitator controls */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-gray-200 bg-white">
        <div className="flex items-center gap-3">
          <MessageSquare className="w-5 h-5 text-primary-600" />
          <h2 className="text-lg font-semibold text-gray-900">Discussion</h2>
          <span className="text-sm text-gray-500">
            {done.length + (currentTopic ? 1 : 0)} / {done.length + (currentTopic ? 1 : 0) + queue.length}
          </span>
        </div>

        <div className="flex items-center gap-3">
          <PhaseTimer isFacilitator={isFacilitator} send={send} />

          {isFacilitator && (
            <>
              <button
                onClick={handleAddTime}
                className="flex items-center gap-1 px-3 py-2 text-sm bg-blue-50 text-blue-700 rounded-lg hover:bg-blue-100 font-medium"
              >
                <Plus className="w-4 h-4" />
                5 min
              </button>
              <button
                onClick={handleNextTopic}
                disabled={queue.length === 0}
                className="flex items-center gap-1 px-3 py-2 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
              >
                <SkipForward className="w-4 h-4" />
                Sujet suivant
              </button>
              {queue.length === 0 && (
                <button
                  onClick={() => send('phase_next', {})}
                  className="flex items-center gap-1 px-3 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium"
                >
                  ROTI
                  <ArrowRight className="w-4 h-4" />
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* 3-column layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left: Queue (À discuter) */}
        <div className="w-64 border-r border-gray-200 bg-gray-50 overflow-auto p-3">
          <h3 className="text-xs font-semibold text-gray-500 uppercase mb-3 flex items-center gap-1">
            <Clock className="w-3 h-3" />
            À discuter ({queue.length})
          </h3>
          <div className="space-y-2">
            {queue.map((topic) => (
              <div
                key={topic.id}
                onClick={() => handleSelectTopic(topic.id)}
                className={clsx(
                  'p-3 bg-white rounded-lg border border-gray-200 shadow-sm',
                  isFacilitator && 'cursor-pointer hover:border-primary-300 hover:shadow'
                )}
              >
                <p className="text-sm text-gray-800 line-clamp-3">{topic.content}</p>
                <div className="flex items-center gap-1 mt-2 text-xs text-gray-500">
                  <ThumbsUp className="w-3 h-3" />
                  {topic.voteCount}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Center: Current topic (En discussion) */}
        <div className="flex-1 overflow-auto p-6">
          {currentTopic ? (
            <div className="max-w-xl mx-auto">
              {/* Current topic card */}
              <div className="bg-white border-2 border-primary-300 rounded-xl p-6 shadow-lg mb-6">
                <div className="flex items-center gap-2 mb-3">
                  <span className="px-2 py-1 bg-primary-100 text-primary-700 rounded text-xs font-medium">
                    En discussion
                  </span>
                  <span className="flex items-center gap-1 text-xs text-gray-500">
                    <ThumbsUp className="w-3 h-3" />
                    {currentTopic.voteCount} votes
                  </span>
                </div>
                <p className="text-lg text-gray-800 whitespace-pre-wrap leading-relaxed">
                  {currentTopic.content}
                </p>
              </div>

              {/* Action creation */}
              <div className="mb-4 p-3 bg-gray-50 rounded-lg border border-gray-200">
                <div className="flex items-center gap-2 mb-3">
                  <Plus className="w-4 h-4 text-primary-600" />
                  <span className="text-sm font-medium text-gray-700">Nouvelle action</span>
                </div>
                <div className="flex gap-2 mb-2">
                  <input
                    type="text"
                    value={newActionTitle}
                    onChange={(e) => setNewActionTitle(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') handleCreateAction() }}
                    placeholder="Titre de l'action..."
                    className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  />
                  <select
                    value={newActionAssignee}
                    onChange={(e) => setNewActionAssignee(e.target.value)}
                    className="px-2 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Non assigné</option>
                    {participants.map((p) => (
                      <option key={p.userId} value={p.userId}>{p.name}</option>
                    ))}
                  </select>
                </div>
                <button
                  onClick={handleCreateAction}
                  disabled={!newActionTitle.trim()}
                  className="w-full px-3 py-1.5 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Créer l'action
                </button>
              </div>

              {/* Actions for current topic */}
              {currentTopicActions.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">
                    Actions ({currentTopicActions.length})
                  </h4>
                  <div className="space-y-2">
                    {currentTopicActions.map((action) => (
                      <div key={action.id} className="flex items-start gap-2 p-2 bg-white border border-gray-200 rounded-lg">
                        <button
                          onClick={() => handleToggleComplete(action)}
                          className={clsx(
                            'mt-0.5 w-4 h-4 border-2 rounded flex-shrink-0 flex items-center justify-center',
                            action.isCompleted ? 'bg-green-500 border-green-500' : 'border-gray-300 hover:border-primary-500'
                          )}
                        >
                          {action.isCompleted && <Check className="w-3 h-3 text-white" />}
                        </button>
                        <div className="flex-1 min-w-0">
                          <p className={clsx('text-xs', action.isCompleted ? 'text-gray-400 line-through' : 'text-gray-900')}>
                            {action.title}
                          </p>
                          {action.assigneeId && (
                            <span className="flex items-center gap-0.5 text-xs text-gray-500 mt-0.5">
                              <User className="w-3 h-3" />
                              {participants.find(p => p.userId === action.assigneeId)?.name || 'Inconnu'}
                            </span>
                          )}
                        </div>
                        <button onClick={() => handleDeleteAction(action.id)} className="p-0.5 text-gray-400 hover:text-red-600 rounded">
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-gray-500">
              <MessageSquare className="w-10 h-10 mb-3 text-gray-300" />
              <p className="text-sm">
                {isFacilitator
                  ? 'Cliquez "Sujet suivant" pour commencer la discussion'
                  : 'En attente du facilitateur...'}
              </p>
            </div>
          )}
        </div>

        {/* Right: Done (Terminé) */}
        <div className="w-64 border-l border-gray-200 bg-gray-50 overflow-auto p-3">
          <h3 className="text-xs font-semibold text-gray-500 uppercase mb-3 flex items-center gap-1">
            <CheckCircle2 className="w-3 h-3" />
            Terminé ({done.length})
          </h3>
          <div className="space-y-2">
            {done.map((topic) => {
              const history = getHistoryForTopic(topic.id)
              return (
                <div key={topic.id} className="p-3 bg-white rounded-lg border border-gray-200 opacity-75">
                  <p className="text-sm text-gray-600 line-clamp-3">{topic.content}</p>
                  <div className="flex items-center gap-2 mt-2 text-xs text-gray-400">
                    <span className="flex items-center gap-0.5">
                      <ThumbsUp className="w-3 h-3" />
                      {topic.voteCount}
                    </span>
                    {history && (
                      <span className="flex items-center gap-0.5">
                        <Clock className="w-3 h-3" />
                        {formatDuration(history.totalDiscussionSeconds)}
                      </span>
                    )}
                  </div>
                </div>
              )
            })}
          </div>

          {/* All actions summary */}
          {actions.length > 0 && (
            <div className="mt-4 pt-4 border-t border-gray-200">
              <h4 className="flex items-center gap-1 text-xs font-medium text-gray-500 uppercase mb-2">
                <ListChecks className="w-3 h-3" />
                Toutes les actions ({actions.length})
              </h4>
              <div className="space-y-1.5">
                {actions.map((action) => (
                  <div key={action.id} className="flex items-center gap-2 px-2 py-1.5 rounded text-xs bg-white border border-gray-100">
                    {action.isCompleted ? (
                      <Check className="w-3 h-3 text-green-500 flex-shrink-0" />
                    ) : (
                      <div className="w-3 h-3 border border-gray-300 rounded-sm flex-shrink-0" />
                    )}
                    <span className={clsx('truncate', action.isCompleted && 'line-through text-gray-400')}>
                      {action.title}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
