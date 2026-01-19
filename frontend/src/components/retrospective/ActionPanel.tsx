import { useState } from 'react'
import { Plus, Check, Trash2, Calendar, User, ChevronDown, ChevronUp } from 'lucide-react'
import type { ActionItem, Participant } from '../../types'

interface ActionPanelProps {
  actions: ActionItem[]
  participants: Participant[]
  canEdit: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

export default function ActionPanel({
  actions,
  participants,
  canEdit,
  send,
}: ActionPanelProps) {
  const [isExpanded, setIsExpanded] = useState(true)
  const [newActionTitle, setNewActionTitle] = useState('')
  const [newActionAssignee, setNewActionAssignee] = useState('')
  const [newActionDueDate, setNewActionDueDate] = useState('')
  const [showForm, setShowForm] = useState(false)

  const handleCreateAction = () => {
    if (!newActionTitle.trim()) return

    send('action_create', {
      title: newActionTitle.trim(),
      assigneeId: newActionAssignee || undefined,
      dueDate: newActionDueDate || undefined,
    })

    setNewActionTitle('')
    setNewActionAssignee('')
    setNewActionDueDate('')
    setShowForm(false)
  }

  const handleToggleComplete = (action: ActionItem) => {
    if (action.isCompleted) {
      send('action_uncomplete', { actionId: action.id })
    } else {
      send('action_complete', { actionId: action.id })
    }
  }

  const handleDeleteAction = (actionId: string) => {
    if (confirm('Supprimer cette action ?')) {
      send('action_delete', { actionId })
    }
  }

  const getParticipantName = (userId: string): string => {
    const participant = participants.find(p => p.userId === userId)
    return participant?.name || 'Non assigné'
  }

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' })
  }

  const pendingActions = actions.filter(a => !a.isCompleted)
  const completedActions = actions.filter(a => a.isCompleted)

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-3 border-b border-gray-200 cursor-pointer"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2">
          <h3 className="font-semibold text-gray-900">Actions</h3>
          <span className="px-2 py-0.5 text-xs bg-primary-100 text-primary-700 rounded-full">
            {pendingActions.length} en cours
          </span>
        </div>
        {isExpanded ? (
          <ChevronUp className="w-5 h-5 text-gray-400" />
        ) : (
          <ChevronDown className="w-5 h-5 text-gray-400" />
        )}
      </div>

      {isExpanded && (
        <div className="p-4">
          {/* Add action button */}
          {canEdit && !showForm && (
            <button
              onClick={() => setShowForm(true)}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm text-primary-600 border border-dashed border-primary-300 rounded-lg hover:bg-primary-50 mb-4"
            >
              <Plus className="w-4 h-4" />
              Ajouter une action
            </button>
          )}

          {/* New action form */}
          {showForm && (
            <div className="mb-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
              <input
                type="text"
                value={newActionTitle}
                onChange={(e) => setNewActionTitle(e.target.value)}
                placeholder="Titre de l'action..."
                className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent mb-3"
                autoFocus
              />

              <div className="flex gap-3 mb-3">
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">Assigné à</label>
                  <select
                    value={newActionAssignee}
                    onChange={(e) => setNewActionAssignee(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Non assigné</option>
                    {participants.map((p) => (
                      <option key={p.userId} value={p.userId}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">Échéance</label>
                  <input
                    type="date"
                    value={newActionDueDate}
                    onChange={(e) => setNewActionDueDate(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setShowForm(false)}
                  className="px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-200 rounded-lg"
                >
                  Annuler
                </button>
                <button
                  onClick={handleCreateAction}
                  disabled={!newActionTitle.trim()}
                  className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
                >
                  Créer
                </button>
              </div>
            </div>
          )}

          {/* Pending actions */}
          {pendingActions.length > 0 && (
            <div className="space-y-2 mb-4">
              {pendingActions.map((action) => (
                <div
                  key={action.id}
                  className="flex items-start gap-3 p-3 bg-white border border-gray-200 rounded-lg hover:shadow-sm"
                >
                  <button
                    onClick={() => handleToggleComplete(action)}
                    className="mt-0.5 w-5 h-5 border-2 border-gray-300 rounded hover:border-primary-500 flex-shrink-0"
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-gray-900">{action.title}</p>
                    <div className="flex items-center gap-3 mt-1 text-xs text-gray-500">
                      {action.assigneeId && (
                        <span className="flex items-center gap-1">
                          <User className="w-3 h-3" />
                          {getParticipantName(action.assigneeId)}
                        </span>
                      )}
                      {action.dueDate && (
                        <span className="flex items-center gap-1">
                          <Calendar className="w-3 h-3" />
                          {formatDate(action.dueDate)}
                        </span>
                      )}
                    </div>
                  </div>
                  {canEdit && (
                    <button
                      onClick={() => handleDeleteAction(action.id)}
                      className="p-1 text-gray-400 hover:text-red-600 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Completed actions */}
          {completedActions.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">
                Terminées ({completedActions.length})
              </h4>
              <div className="space-y-2">
                {completedActions.map((action) => (
                  <div
                    key={action.id}
                    className="flex items-start gap-3 p-3 bg-gray-50 border border-gray-100 rounded-lg"
                  >
                    <button
                      onClick={() => handleToggleComplete(action)}
                      className="mt-0.5 w-5 h-5 bg-green-500 border-2 border-green-500 rounded flex items-center justify-center flex-shrink-0"
                    >
                      <Check className="w-3 h-3 text-white" />
                    </button>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-gray-500 line-through">{action.title}</p>
                    </div>
                    {canEdit && (
                      <button
                        onClick={() => handleDeleteAction(action.id)}
                        className="p-1 text-gray-400 hover:text-red-600 rounded"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Empty state */}
          {actions.length === 0 && !showForm && (
            <p className="text-sm text-gray-500 text-center py-4">
              Aucune action pour le moment
            </p>
          )}
        </div>
      )}
    </div>
  )
}
