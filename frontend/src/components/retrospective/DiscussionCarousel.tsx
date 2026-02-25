import { useState, useMemo, useEffect } from 'react'
import { ChevronLeft, ChevronRight, ThumbsUp, MessageSquare, Layers, Plus, Check, Trash2, Calendar, User, ArrowRight, ListChecks } from 'lucide-react'
import type { ActionItem, Item, Participant, Template } from '../../types'
import clsx from 'clsx'

interface DiscussionCarouselProps {
  items: Item[]
  template: Template
  getAuthorName: (authorId: string) => string
  actions: ActionItem[]
  participants: Participant[]
  send: (type: string, payload: Record<string, unknown>) => void
  isFacilitator: boolean
  syncItemId?: string | null  // externally controlled item (from discuss_item_changed)
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

export default function DiscussionCarousel({
  items,
  template,
  getAuthorName,
  actions,
  participants,
  send,
  isFacilitator,
  syncItemId,
}: DiscussionCarouselProps) {
  const [localIndex, setLocalIndex] = useState(0)
  const [newActionTitle, setNewActionTitle] = useState('')
  const [newActionAssignee, setNewActionAssignee] = useState('')
  const [newActionDueDate, setNewActionDueDate] = useState('')

  // currentIndex is controlled: facilitator sends discuss_set_item, all clients sync
  const currentIndex = localIndex

  // Get top-level items (not grouped under another item), sorted by votes
  const discussionItems = useMemo(() => {
    const topLevelItems = items.filter(item => !item.groupId)
    return topLevelItems.sort((a, b) => {
      const aGroupedItems = items.filter(i => i.groupId === a.id)
      const bGroupedItems = items.filter(i => i.groupId === b.id)
      const aTotalVotes = a.voteCount + aGroupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      const bTotalVotes = b.voteCount + bGroupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      return bTotalVotes - aTotalVotes
    })
  }, [items])

  const getGroupedItems = (parentId: string): Item[] => {
    return items.filter(item => item.groupId === parentId)
  }

  const getColumnInfo = (columnId: string) => {
    return template.columns.find(c => c.id === columnId)
  }

  const getTotalVotes = (item: Item): number => {
    const groupedItems = getGroupedItems(item.id)
    return item.voteCount + groupedItems.reduce((sum, i) => sum + i.voteCount, 0)
  }

  const currentItem = discussionItems[currentIndex]
  const groupedItems = currentItem ? getGroupedItems(currentItem.id) : []
  const totalItems = discussionItems.length

  const currentItemActions = currentItem
    ? actions.filter(a => a.itemId === currentItem.id)
    : []

  const goToPrevious = () => {
    const newIndex = currentIndex > 0 ? currentIndex - 1 : totalItems - 1
    if (isFacilitator && discussionItems[newIndex]) {
      send('discuss_set_item', { itemId: discussionItems[newIndex].id })
    }
    setLocalIndex(newIndex)
  }

  const goToNext = () => {
    const newIndex = currentIndex < totalItems - 1 ? currentIndex + 1 : 0
    if (isFacilitator && discussionItems[newIndex]) {
      send('discuss_set_item', { itemId: discussionItems[newIndex].id })
    }
    setLocalIndex(newIndex)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    const target = e.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'SELECT' || target.tagName === 'TEXTAREA') {
      return
    }
    if (e.key === 'ArrowLeft') goToPrevious()
    if (e.key === 'ArrowRight') goToNext()
  }

  const handleCreateAction = () => {
    if (!newActionTitle.trim()) return

    send('action_create', {
      title: newActionTitle.trim(),
      assigneeId: newActionAssignee || undefined,
      dueDate: newActionDueDate || undefined,
      itemId: currentItem?.id,
    })

    setNewActionTitle('')
    setNewActionAssignee('')
    setNewActionDueDate('')
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

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' })
  }

  const getParticipantName = (userId: string): string => {
    const participant = participants.find(p => p.userId === userId)
    return participant?.name || 'Inconnu'
  }

  if (!currentItem) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500">
        Aucun item à discuter
      </div>
    )
  }

  const column = getColumnInfo(currentItem.columnId)
  const totalVotes = getTotalVotes(currentItem)

  return (
    <div
      className="flex flex-col h-full"
      onKeyDown={handleKeyDown}
      tabIndex={0}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-gray-200 bg-white">
        <div className="flex items-center gap-3">
          <MessageSquare className="w-5 h-5 text-primary-600" />
          <h2 className="text-lg font-semibold text-gray-900">Discussion</h2>
          <span className="text-sm text-gray-500">
            {currentIndex + 1} / {totalItems}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {isFacilitator && (
            <button
              onClick={() => send('phase_next', {})}
              className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-green-600 text-white hover:bg-green-700"
            >
              Continuer vers ROTI
              <ArrowRight className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Content - 2 column layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left column - Discussion item */}
        <div className="flex-1 min-w-0 overflow-auto p-6">
          {/* Vote count badge */}
          <div className="flex items-center justify-center mb-6">
            <div className="flex items-center gap-2 px-4 py-2 bg-primary-50 text-primary-700 rounded-full">
              <ThumbsUp className="w-5 h-5" />
              <span className="text-lg font-semibold">{totalVotes} vote{totalVotes !== 1 ? 's' : ''}</span>
            </div>
          </div>

          {/* Main item card */}
          <div
            className="bg-white border-2 rounded-xl p-6 mb-4 max-w-2xl mx-auto"
            style={{ borderColor: column?.color || '#e5e7eb' }}
          >
            {/* Labels */}
            <div className="flex flex-wrap gap-2 mb-4">
              <span
                className="inline-flex items-center px-2 py-1 rounded text-sm font-medium"
                style={{ backgroundColor: `${column?.color}20`, color: column?.color }}
              >
                {column?.name || currentItem.columnId}
              </span>
              <span className="inline-flex items-center px-2 py-1 rounded text-sm font-medium bg-gray-100 text-gray-700">
                {getTrigram(getAuthorName(currentItem.authorId))}
              </span>
              <span className="inline-flex items-center px-2 py-1 rounded text-sm font-medium bg-blue-50 text-blue-700">
                <ThumbsUp className="w-3 h-3 mr-1" />
                {currentItem.voteCount}
              </span>
            </div>

            {/* Content */}
            <p className="text-lg text-gray-800 whitespace-pre-wrap leading-relaxed">
              {currentItem.content}
            </p>
          </div>

          {/* Grouped items */}
          {groupedItems.length > 0 && (
            <div className="mt-6 max-w-2xl mx-auto">
              <div className="flex items-center gap-2 mb-3 text-gray-600">
                <Layers className="w-4 h-4" />
                <span className="text-sm font-medium">
                  {groupedItems.length} item{groupedItems.length > 1 ? 's' : ''} groupé{groupedItems.length > 1 ? 's' : ''}
                </span>
              </div>
              <div className="space-y-3 pl-4 border-l-2 border-gray-200">
                {groupedItems.map(grouped => {
                  const groupedColumn = getColumnInfo(grouped.columnId)
                  return (
                    <div
                      key={grouped.id}
                      className="bg-gray-50 rounded-lg p-4 border border-gray-200"
                    >
                      <div className="flex flex-wrap gap-2 mb-2">
                        <span
                          className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium"
                          style={{ backgroundColor: `${groupedColumn?.color}20`, color: groupedColumn?.color }}
                        >
                          {groupedColumn?.name || grouped.columnId}
                        </span>
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                          {getTrigram(getAuthorName(grouped.authorId))}
                        </span>
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">
                          <ThumbsUp className="w-3 h-3 mr-1" />
                          {grouped.voteCount}
                        </span>
                      </div>
                      <p className="text-sm text-gray-700 whitespace-pre-wrap">
                        {grouped.content}
                      </p>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        {/* Right column - Actions panel */}
        <div className="w-80 flex flex-col border-l border-gray-200 bg-white overflow-auto">
          <div className="p-4">
            {/* Action form */}
            <div className="mb-4 p-3 bg-gray-50 rounded-lg border border-gray-200">
              <div className="flex items-center gap-2 mb-3">
                <Plus className="w-4 h-4 text-primary-600" />
                <span className="text-sm font-medium text-gray-700">Nouvelle action</span>
              </div>

              <input
                type="text"
                value={newActionTitle}
                onChange={(e) => setNewActionTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCreateAction()
                }}
                placeholder="Titre de l'action..."
                className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent mb-2"
              />

              <div className="grid grid-cols-2 gap-2 mb-2">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Assigné</label>
                  <select
                    value={newActionAssignee}
                    onChange={(e) => setNewActionAssignee(e.target.value)}
                    className="w-full px-2 py-1.5 text-xs border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
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
                    className="w-full px-2 py-1.5 text-xs border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  />
                </div>
              </div>

              <button
                onClick={handleCreateAction}
                disabled={!newActionTitle.trim()}
                className="w-full px-3 py-1.5 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Créer l'action
              </button>
            </div>

            {/* Actions for current item */}
            {currentItemActions.length > 0 && (
              <div className="mb-4">
                <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">
                  Actions de cet item ({currentItemActions.length})
                </h4>
                <div className="space-y-2">
                  {currentItemActions.map((action) => (
                    <div
                      key={action.id}
                      className="flex items-start gap-2 p-2 bg-white border border-gray-200 rounded-lg"
                    >
                      <button
                        onClick={() => handleToggleComplete(action)}
                        className={clsx(
                          'mt-0.5 w-4 h-4 border-2 rounded flex-shrink-0 flex items-center justify-center',
                          action.isCompleted
                            ? 'bg-green-500 border-green-500'
                            : 'border-gray-300 hover:border-primary-500'
                        )}
                      >
                        {action.isCompleted && <Check className="w-3 h-3 text-white" />}
                      </button>
                      <div className="flex-1 min-w-0">
                        <p className={clsx('text-xs', action.isCompleted ? 'text-gray-400 line-through' : 'text-gray-900')}>
                          {action.title}
                        </p>
                        <div className="flex items-center gap-2 mt-0.5 text-xs text-gray-500">
                          {action.assigneeId && (
                            <span className="flex items-center gap-0.5">
                              <User className="w-3 h-3" />
                              {getParticipantName(action.assigneeId)}
                            </span>
                          )}
                          {action.dueDate && (
                            <span className="flex items-center gap-0.5">
                              <Calendar className="w-3 h-3" />
                              {formatDate(action.dueDate)}
                            </span>
                          )}
                        </div>
                      </div>
                      <button
                        onClick={() => handleDeleteAction(action.id)}
                        className="p-0.5 text-gray-400 hover:text-red-600 rounded"
                      >
                        <Trash2 className="w-3 h-3" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* All actions summary */}
            {actions.length > 0 && (
              <div>
                <h4 className="flex items-center gap-1 text-xs font-medium text-gray-500 uppercase mb-2">
                  <ListChecks className="w-3 h-3" />
                  Toutes les actions ({actions.length})
                </h4>
                <div className="space-y-1.5 max-h-48 overflow-auto">
                  {actions.map((action) => (
                    <div
                      key={action.id}
                      className={clsx(
                        'flex items-center gap-2 px-2 py-1.5 rounded text-xs',
                        action.itemId === currentItem?.id
                          ? 'bg-primary-50 border border-primary-200'
                          : 'bg-gray-50 border border-gray-100'
                      )}
                    >
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

            {actions.length === 0 && currentItemActions.length === 0 && (
              <div className="text-center py-6 text-gray-500 text-xs">
                <p>Aucune action créée</p>
                <p className="mt-1">Créez des actions pendant la discussion</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 bg-white">
        <button
          onClick={goToPrevious}
          className={clsx(
            'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors',
            'bg-white border border-gray-300 text-gray-700 hover:bg-gray-50'
          )}
        >
          <ChevronLeft className="w-5 h-5" />
          Précédent
        </button>

        {/* Progress dots */}
        <div className="flex items-center gap-1.5">
          {discussionItems.slice(0, 10).map((item, index) => (
            <button
              key={index}
              onClick={() => {
                if (isFacilitator) {
                  send('discuss_set_item', { itemId: item.id })
                }
                setLocalIndex(index)
              }}
              className={clsx(
                'w-2.5 h-2.5 rounded-full transition-colors',
                index === currentIndex
                  ? 'bg-primary-600'
                  : 'bg-gray-300 hover:bg-gray-400'
              )}
            />
          ))}
          {totalItems > 10 && (
            <span className="text-xs text-gray-500 ml-1">+{totalItems - 10}</span>
          )}
        </div>

        <button
          onClick={goToNext}
          className={clsx(
            'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors',
            'bg-primary-600 text-white hover:bg-primary-700'
          )}
        >
          Suivant
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    </div>
  )
}
