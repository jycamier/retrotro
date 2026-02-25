import { useState } from 'react'
import { useRetroStore } from '../../store/retroStore'
import { useAuthStore } from '../../store/authStore'
import { Plus, MessageSquare, Trash2 } from 'lucide-react'

interface LCProposePhaseViewProps {
  send: (type: string, payload: Record<string, unknown>) => void
  isFacilitator: boolean
}

export default function LCProposePhaseView({ send, isFacilitator }: LCProposePhaseViewProps) {
  const { items } = useRetroStore()
  const { user } = useAuthStore()
  const [newTopic, setNewTopic] = useState('')

  // LC items use a single column 'topics'
  const topics = items.filter(item => !item.groupId)

  const handleSubmit = () => {
    if (!newTopic.trim()) return
    send('item_create', { columnId: 'topics', content: newTopic.trim() })
    setNewTopic('')
  }

  const handleDelete = (itemId: string) => {
    send('item_delete', { itemId })
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="text-center mb-6">
        <MessageSquare className="w-8 h-8 text-primary-600 mx-auto mb-2" />
        <h2 className="text-xl font-semibold text-gray-900">Proposez vos sujets</h2>
        <p className="text-sm text-gray-500 mt-1">
          Écrivez les sujets que vous aimeriez discuter
        </p>
      </div>

      {/* Input */}
      <div className="mb-6">
        <div className="flex gap-2">
          <input
            type="text"
            value={newTopic}
            onChange={(e) => setNewTopic(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleSubmit() }}
            placeholder="Proposer un sujet..."
            className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent text-sm"
            autoFocus
          />
          <button
            onClick={handleSubmit}
            disabled={!newTopic.trim()}
            className="flex items-center gap-2 px-4 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Plus className="w-4 h-4" />
            Ajouter
          </button>
        </div>
      </div>

      {/* Topic list */}
      <div className="space-y-3">
        {topics.length === 0 ? (
          <div className="text-center py-8 text-gray-400 text-sm">
            Aucun sujet proposé pour le moment
          </div>
        ) : (
          topics.map((topic) => (
            <div
              key={topic.id}
              className="flex items-start gap-3 p-4 bg-white rounded-lg border border-gray-200 shadow-sm"
            >
              <div className="flex-1">
                <p className="text-sm text-gray-800 whitespace-pre-wrap">{topic.content}</p>
                <p className="text-xs text-gray-400 mt-1">
                  {topic.authorId === user?.id ? 'Vous' : 'Participant'}
                </p>
              </div>
              {topic.authorId === user?.id && (
                <button
                  onClick={() => handleDelete(topic.id)}
                  className="p-1 text-gray-400 hover:text-red-500 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))
        )}
      </div>

      <div className="mt-4 text-center text-sm text-gray-500">
        {topics.length} sujet{topics.length !== 1 ? 's' : ''} proposé{topics.length !== 1 ? 's' : ''}
      </div>

      {isFacilitator && (
        <div className="mt-6 text-center">
          <button
            onClick={() => send('phase_next', {})}
            className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 text-sm font-medium"
          >
            Passer au vote
          </button>
        </div>
      )}
    </div>
  )
}
