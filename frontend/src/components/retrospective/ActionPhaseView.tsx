import { useState } from 'react'
import { Plus, Check, Trash2, Calendar, User, ThumbsUp, ArrowRight, Layers } from 'lucide-react'
import type { ActionItem, Item, Participant, Template } from '../../types'
import clsx from 'clsx'

interface ActionPhaseViewProps {
  items: Item[]
  actions: ActionItem[]
  participants: Participant[]
  template: Template
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

// Helper to get trigram from name
const getTrigram = (name: string): string => {
  if (!name) return '???'
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 3) {
    return parts.slice(0, 3).map(p => p[0]?.toUpperCase() || '').join('')
  } else if (parts.length === 2) {
    return (parts[0][0] + parts[1].slice(0, 2)).toUpperCase()
  }
  return name.slice(0, 3).toUpperCase()
}

export default function ActionPhaseView({
  items,
  actions,
  participants,
  template,
  isFacilitator,
  send,
}: ActionPhaseViewProps) {
  const [selectedItem, setSelectedItem] = useState<Item | null>(null)
  const [newActionTitle, setNewActionTitle] = useState('')
  const [newActionAssignee, setNewActionAssignee] = useState('')
  const [newActionDueDate, setNewActionDueDate] = useState('')

  // Get top-level items sorted by votes
  const sortedItems = items
    .filter(item => !item.groupId)
    .map(item => {
      const groupedItems = items.filter(i => i.groupId === item.id)
      const totalVotes = item.voteCount + groupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      return { ...item, totalVotes, groupedItems }
    })
    .sort((a, b) => b.totalVotes - a.totalVotes)

  const getColumnInfo = (columnId: string) => {
    return template.columns.find(c => c.id === columnId)
  }

  const getParticipantName = (userId: string): string => {
    const participant = participants.find(p => p.userId === userId)
    if (participant) {
      return participant.name
    }
    // If no participants yet, show loading indicator
    if (participants.length === 0) {
      return '...'
    }
    return 'Inconnu'
  }

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' })
  }

  const handleCreateAction = () => {
    if (!newActionTitle.trim()) return

    send('action_create', {
      title: newActionTitle.trim(),
      assigneeId: newActionAssignee || undefined,
      dueDate: newActionDueDate || undefined,
      itemId: selectedItem?.id,
    })

    setNewActionTitle('')
    setNewActionAssignee('')
    setNewActionDueDate('')
    setSelectedItem(null)
  }

  const handleCreateFromItem = (item: Item) => {
    setSelectedItem(item)
    setNewActionTitle(item.content.length > 100 ? item.content.substring(0, 100) + '...' : item.content)
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

  const pendingActions = actions.filter(a => !a.isCompleted)
  const completedActions = actions.filter(a => a.isCompleted)

  return (
    <div className="flex gap-6 h-full">
      {/* Left column - Items to discuss */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">
            Points à traiter
          </h2>
          <span className="text-sm text-gray-500">
            {sortedItems.length} item{sortedItems.length > 1 ? 's' : ''}
          </span>
        </div>

        <div className="flex-1 overflow-auto space-y-3 pr-2">
          {sortedItems.map((item, index) => {
            const column = getColumnInfo(item.columnId)
            const isSelected = selectedItem?.id === item.id

            return (
              <div
                key={item.id}
                className={clsx(
                  'bg-white rounded-lg border-2 p-4 transition-all cursor-pointer',
                  isSelected
                    ? 'border-primary-500 ring-2 ring-primary-200'
                    : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
                )}
                onClick={() => setSelectedItem(isSelected ? null : item)}
              >
                <div className="flex items-start gap-3">
                  {/* Rank */}
                  <div className={clsx(
                    'flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold',
                    index === 0 ? 'bg-yellow-100 text-yellow-700' :
                    index === 1 ? 'bg-gray-100 text-gray-600' :
                    index === 2 ? 'bg-orange-100 text-orange-700' :
                    'bg-gray-50 text-gray-500'
                  )}>
                    {index + 1}
                  </div>

                  <div className="flex-1 min-w-0">
                    {/* Labels */}
                    <div className="flex flex-wrap gap-2 mb-2">
                      <span
                        className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
                        style={{ backgroundColor: `${column?.color}20`, color: column?.color }}
                      >
                        {column?.name}
                      </span>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                        {getTrigram(getParticipantName(item.authorId))}
                      </span>
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-primary-50 text-primary-700">
                        <ThumbsUp className="w-3 h-3" />
                        {item.totalVotes}
                      </span>
                      {item.groupedItems.length > 0 && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                          <Layers className="w-3 h-3" />
                          +{item.groupedItems.length}
                        </span>
                      )}
                    </div>

                    {/* Content */}
                    <p className="text-sm text-gray-800 whitespace-pre-wrap">
                      {item.content}
                    </p>

                    {/* Grouped items preview */}
                    {item.groupedItems.length > 0 && (
                      <div className="mt-2 pl-3 border-l-2 border-gray-200 space-y-1">
                        {item.groupedItems.slice(0, 2).map(grouped => (
                          <p key={grouped.id} className="text-xs text-gray-500 truncate">
                            {grouped.content}
                          </p>
                        ))}
                        {item.groupedItems.length > 2 && (
                          <p className="text-xs text-gray-400">
                            +{item.groupedItems.length - 2} autres...
                          </p>
                        )}
                      </div>
                    )}
                  </div>

                  {/* Create action button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleCreateFromItem(item)
                    }}
                    className="flex-shrink-0 p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors"
                    title="Créer une action"
                  >
                    <ArrowRight className="w-5 h-5" />
                  </button>
                </div>
              </div>
            )
          })}

          {sortedItems.length === 0 && (
            <div className="text-center py-8 text-gray-500">
              Aucun item à traiter
            </div>
          )}
        </div>
      </div>

      {/* Right column - Actions */}
      <div className="w-96 flex flex-col bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50 rounded-t-lg">
          <h2 className="text-lg font-semibold text-gray-900">
            Actions
          </h2>
          <p className="text-sm text-gray-500">
            {pendingActions.length} en cours, {completedActions.length} terminée{completedActions.length > 1 ? 's' : ''}
          </p>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {/* New action form */}
          <div className="mb-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
            <div className="flex items-center gap-2 mb-3">
              <Plus className="w-4 h-4 text-primary-600" />
              <span className="text-sm font-medium text-gray-700">Nouvelle action</span>
            </div>

            {selectedItem && (
              <div className="mb-3 p-2 bg-white rounded border border-primary-200 text-xs">
                <span className="text-gray-500">Depuis : </span>
                <span className="text-gray-700">{selectedItem.content.substring(0, 50)}...</span>
                <button
                  onClick={() => setSelectedItem(null)}
                  className="ml-2 text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </div>
            )}

            <input
              type="text"
              value={newActionTitle}
              onChange={(e) => setNewActionTitle(e.target.value)}
              placeholder="Titre de l'action..."
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent mb-3"
            />

            <div className="grid grid-cols-2 gap-2 mb-3">
              <div>
                <label className="block text-xs text-gray-500 mb-1">Assigné</label>
                <select
                  value={newActionAssignee}
                  onChange={(e) => setNewActionAssignee(e.target.value)}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                >
                  <option value="">Non assigné</option>
                  {participants.map((p) => (
                    <option key={p.userId} value={p.userId}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Échéance</label>
                <input
                  type="date"
                  value={newActionDueDate}
                  onChange={(e) => setNewActionDueDate(e.target.value)}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>

            <button
              onClick={handleCreateAction}
              disabled={!newActionTitle.trim()}
              className="w-full px-3 py-2 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Créer l'action
            </button>
          </div>

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
                  <button
                    onClick={() => handleDeleteAction(action.id)}
                    className="p-1 text-gray-400 hover:text-red-600 rounded"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
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
                    <button
                      onClick={() => handleDeleteAction(action.id)}
                      className="p-1 text-gray-400 hover:text-red-600 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Empty state */}
          {actions.length === 0 && (
            <div className="text-center py-8 text-gray-500 text-sm">
              <p>Aucune action créée</p>
              <p className="mt-1 text-xs">Cliquez sur un item à gauche pour créer une action</p>
            </div>
          )}
        </div>

        {/* Next Phase Button - Facilitator only */}
        {isFacilitator && (
          <div className="px-4 py-3 border-t border-gray-200 bg-gray-50 rounded-b-lg">
            <button
              onClick={() => send('phase_next', {})}
              className="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
            >
              Continuer vers ROTI
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
