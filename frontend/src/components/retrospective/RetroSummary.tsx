import { Download, CheckCircle, Users, ThumbsUp, Calendar, User, ExternalLink } from 'lucide-react'
import type { Retrospective, Item, ActionItem, Participant, Template } from '../../types'

interface RetroSummaryProps {
  retro: Retrospective
  items: Item[]
  actions: ActionItem[]
  participants: Participant[]
  template: Template
  onExport?: () => void
  onClose: () => void
}

export default function RetroSummary({
  retro,
  items,
  actions,
  participants,
  template,
  onExport,
  onClose,
}: RetroSummaryProps) {
  // Get top items by votes
  const topItems = items
    .filter(item => !item.groupId)
    .map(item => {
      const groupedItems = items.filter(i => i.groupId === item.id)
      const totalVotes = item.voteCount + groupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      return { ...item, totalVotes }
    })
    .sort((a, b) => b.totalVotes - a.totalVotes)
    .slice(0, 5)

  const getColumnInfo = (columnId: string) => {
    return template.columns.find(c => c.id === columnId)
  }

  const getParticipantName = (userId: string): string => {
    const participant = participants.find(p => p.userId === userId)
    return participant?.name || 'Non assigné'
  }

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
  }

  const pendingActions = actions.filter(a => !a.isCompleted)
  const completedActions = actions.filter(a => a.isCompleted)
  const totalVotes = items.reduce((sum, item) => sum + item.voteCount, 0)

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="bg-gradient-to-r from-primary-600 to-primary-700 px-6 py-8 text-white">
          <div className="flex items-center gap-3 mb-2">
            <CheckCircle className="w-8 h-8" />
            <h1 className="text-2xl font-bold">Rétrospective terminée</h1>
          </div>
          <p className="text-primary-100 text-lg">{retro.name}</p>
          <p className="text-primary-200 text-sm mt-1">
            Terminée le {formatDate(retro.endedAt || new Date().toISOString())}
          </p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4 px-6 py-4 bg-gray-50 border-b">
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{participants.length}</div>
            <div className="text-sm text-gray-500 flex items-center justify-center gap-1">
              <Users className="w-4 h-4" />
              Participants
            </div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{items.filter(i => !i.groupId).length}</div>
            <div className="text-sm text-gray-500">Items créés</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{totalVotes}</div>
            <div className="text-sm text-gray-500 flex items-center justify-center gap-1">
              <ThumbsUp className="w-4 h-4" />
              Votes
            </div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-primary-600">{actions.length}</div>
            <div className="text-sm text-gray-500">Actions</div>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          <div className="grid grid-cols-2 gap-6">
            {/* Top items */}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Top 5 des points discutés
              </h2>
              <div className="space-y-3">
                {topItems.map((item, index) => {
                  const column = getColumnInfo(item.columnId)
                  return (
                    <div
                      key={item.id}
                      className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg"
                    >
                      <div className={`
                        flex-shrink-0 w-7 h-7 rounded-full flex items-center justify-center text-sm font-bold
                        ${index === 0 ? 'bg-yellow-100 text-yellow-700' :
                          index === 1 ? 'bg-gray-200 text-gray-600' :
                          index === 2 ? 'bg-orange-100 text-orange-700' :
                          'bg-gray-100 text-gray-500'}
                      `}>
                        {index + 1}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span
                            className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
                            style={{ backgroundColor: `${column?.color}20`, color: column?.color }}
                          >
                            {column?.name}
                          </span>
                          <span className="text-xs text-gray-500 flex items-center gap-1">
                            <ThumbsUp className="w-3 h-3" />
                            {item.totalVotes}
                          </span>
                        </div>
                        <p className="text-sm text-gray-700 line-clamp-2">{item.content}</p>
                      </div>
                    </div>
                  )
                })}
                {topItems.length === 0 && (
                  <p className="text-sm text-gray-500 text-center py-4">
                    Aucun item dans cette rétrospective
                  </p>
                )}
              </div>
            </div>

            {/* Actions */}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Actions à suivre ({pendingActions.length})
              </h2>
              <div className="space-y-2">
                {pendingActions.map((action) => (
                  <div
                    key={action.id}
                    className="flex items-start gap-3 p-3 bg-primary-50 border border-primary-100 rounded-lg"
                  >
                    <div className="w-5 h-5 border-2 border-primary-300 rounded flex-shrink-0 mt-0.5" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-gray-900 font-medium">{action.title}</p>
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
                  </div>
                ))}
                {pendingActions.length === 0 && (
                  <p className="text-sm text-gray-500 text-center py-4">
                    Aucune action en attente
                  </p>
                )}
              </div>

              {completedActions.length > 0 && (
                <div className="mt-4">
                  <h3 className="text-sm font-medium text-gray-500 mb-2">
                    Actions déjà terminées ({completedActions.length})
                  </h3>
                  <div className="space-y-1">
                    {completedActions.slice(0, 3).map((action) => (
                      <div
                        key={action.id}
                        className="flex items-center gap-2 p-2 bg-gray-50 rounded text-sm text-gray-500"
                      >
                        <CheckCircle className="w-4 h-4 text-green-500" />
                        <span className="line-through">{action.title}</span>
                      </div>
                    ))}
                    {completedActions.length > 3 && (
                      <p className="text-xs text-gray-400 pl-6">
                        +{completedActions.length - 3} autres...
                      </p>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 bg-gray-50 border-t flex items-center justify-between">
          <div className="flex items-center gap-2">
            {onExport && (
              <button
                onClick={onExport}
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                <Download className="w-4 h-4" />
                Exporter
              </button>
            )}
          </div>
          <button
            onClick={onClose}
            className="flex items-center gap-2 px-6 py-2 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700"
          >
            Fermer
            <ExternalLink className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
