import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { retrosApi, templatesApi } from '../api/client'
import { Play, Trash2, ArrowLeft, CheckCircle, ThumbsUp, Calendar, User, Star, Sun, CloudSun, Cloud, CloudRain, CloudLightning } from 'lucide-react'
import type { Item, ActionItem, MoodWeather, IcebreakerMood } from '../types'

// Mood config for display
const moodConfig: Record<MoodWeather, { icon: typeof Sun; label: string; color: string; bgColor: string }> = {
  sunny: { icon: Sun, label: 'Ensoleillé', color: 'text-yellow-500', bgColor: 'bg-yellow-50 border-yellow-200' },
  partly_cloudy: { icon: CloudSun, label: 'Partiellement nuageux', color: 'text-orange-400', bgColor: 'bg-orange-50 border-orange-200' },
  cloudy: { icon: Cloud, label: 'Nuageux', color: 'text-gray-400', bgColor: 'bg-gray-50 border-gray-200' },
  rainy: { icon: CloudRain, label: 'Pluvieux', color: 'text-blue-400', bgColor: 'bg-blue-50 border-blue-200' },
  stormy: { icon: CloudLightning, label: 'Orageux', color: 'text-purple-500', bgColor: 'bg-purple-50 border-purple-200' },
}

export default function RetroPage() {
  const { teamId, retroId } = useParams<{ teamId: string; retroId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: retro, isLoading } = useQuery({
    queryKey: ['retro', retroId],
    queryFn: () => retrosApi.get(retroId!),
    enabled: !!retroId,
  })

  const { data: template } = useQuery({
    queryKey: ['template', retro?.templateId],
    queryFn: () => templatesApi.get(retro!.templateId),
    enabled: !!retro?.templateId,
  })

  // Fetch items and actions for completed retros
  const { data: items } = useQuery({
    queryKey: ['retro-items', retroId],
    queryFn: () => retrosApi.getItems(retroId!),
    enabled: !!retroId && retro?.status === 'completed',
  })

  const { data: actions } = useQuery({
    queryKey: ['retro-actions', retroId],
    queryFn: () => retrosApi.getActions(retroId!),
    enabled: !!retroId && retro?.status === 'completed',
  })

  // Fetch icebreaker moods for completed retros
  const { data: icebreakerMoods } = useQuery({
    queryKey: ['retro-icebreaker', retroId],
    queryFn: () => retrosApi.getIcebreakerMoods(retroId!),
    enabled: !!retroId && retro?.status === 'completed',
  })

  // Fetch ROTI results for completed retros
  const { data: rotiResults } = useQuery({
    queryKey: ['roti', retroId],
    queryFn: () => retrosApi.getRotiResults(retroId!),
    enabled: !!retroId && retro?.status === 'completed',
  })

  const startMutation = useMutation({
    mutationFn: () => retrosApi.start(retroId!),
    onSuccess: () => {
      navigate(`/retro/${retroId}`)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => retrosApi.delete(retroId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['retros', teamId] })
      navigate(`/teams/${teamId}`)
    },
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  if (!retro) {
    return <div>Retrospective not found</div>
  }

  const getColumnInfo = (columnId: string) => {
    return template?.columns.find(c => c.id === columnId)
  }

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
  }

  // Get initials from user display name
  const getInitials = (name: string): string => {
    if (!name) return '??'
    const parts = name.trim().split(/\s+/)
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase()
    }
    return name.slice(0, 2).toUpperCase()
  }

  // For completed retros, show the summary view
  if (retro.status === 'completed' && items && actions) {
    const topItems = items
      .filter((item: Item) => !item.groupId)
      .map((item: Item) => {
        const groupedItems = items.filter((i: Item) => i.groupId === item.id)
        const totalVotes = item.voteCount + groupedItems.reduce((sum: number, i: Item) => sum + i.voteCount, 0)
        return { ...item, totalVotes }
      })
      .sort((a, b) => b.totalVotes - a.totalVotes)
      .slice(0, 10)

    const pendingActions = actions.filter((a: ActionItem) => !a.isCompleted)
    const completedActions = actions.filter((a: ActionItem) => a.isCompleted)
    const totalVotes = items.reduce((sum: number, item: Item) => sum + item.voteCount, 0)

    return (
      <div className="space-y-6">
        <button
          onClick={() => navigate(`/teams/${teamId}`)}
          className="flex items-center gap-2 text-gray-600 hover:text-gray-900"
        >
          <ArrowLeft className="w-4 h-4" />
          Retour à l'équipe
        </button>

        {/* Header */}
        <div className="bg-gradient-to-r from-primary-600 to-primary-700 rounded-lg px-6 py-8 text-white">
          <div className="flex items-center gap-3 mb-2">
            <CheckCircle className="w-8 h-8" />
            <h1 className="text-2xl font-bold">Rétrospective terminée</h1>
          </div>
          <p className="text-primary-100 text-lg">{retro.name}</p>
          <p className="text-primary-200 text-sm mt-1">
            Terminée le {formatDate(retro.endedAt || retro.updatedAt)}
          </p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-5 gap-4">
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-gray-900">{items.filter((i: Item) => !i.groupId).length}</div>
            <div className="text-sm text-gray-500">Items créés</div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-gray-900">{totalVotes}</div>
            <div className="text-sm text-gray-500 flex items-center justify-center gap-1">
              <ThumbsUp className="w-4 h-4" />
              Votes
            </div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-primary-600">{pendingActions.length}</div>
            <div className="text-sm text-gray-500">Actions en cours</div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-green-600">{completedActions.length}</div>
            <div className="text-sm text-gray-500">Actions terminées</div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            {rotiResults && rotiResults.totalVotes > 0 ? (
              <>
                <div className="flex items-center justify-center gap-1">
                  <div className="text-2xl font-bold text-yellow-600">{rotiResults.average.toFixed(1)}</div>
                  <Star className="w-5 h-5 text-yellow-500 fill-yellow-500" />
                </div>
                <div className="text-sm text-gray-500">ROTI ({rotiResults.totalVotes} votes)</div>
              </>
            ) : (
              <>
                <div className="text-2xl font-bold text-gray-400">-</div>
                <div className="text-sm text-gray-500">ROTI</div>
              </>
            )}
          </div>
        </div>

        {/* Icebreaker Moods */}
        {icebreakerMoods && icebreakerMoods.length > 0 && (
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Humeur de l'équipe
            </h2>
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {icebreakerMoods.map((moodEntry: IcebreakerMood) => {
                const config = moodConfig[moodEntry.mood]
                const Icon = config.icon
                const displayName = moodEntry.user?.displayName || 'Inconnu'

                return (
                  <div
                    key={moodEntry.id}
                    className={`flex items-center gap-2 p-3 rounded-lg border ${config.bgColor}`}
                  >
                    <div className="w-8 h-8 rounded-full bg-white flex items-center justify-center text-xs font-medium text-gray-700 shadow-sm">
                      {getInitials(displayName)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 truncate">
                        {displayName}
                      </p>
                    </div>
                    <Icon className={`w-5 h-5 flex-shrink-0 ${config.color}`} />
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Content */}
        <div className="grid grid-cols-2 gap-6">
          {/* Top items */}
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Top des points discutés
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
                      <p className="text-sm text-gray-700">{item.content}</p>
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
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Actions ({actions.length})
            </h2>

            {/* Pending actions */}
            {pendingActions.length > 0 && (
              <div className="space-y-2 mb-4">
                <h3 className="text-sm font-medium text-gray-500">En cours ({pendingActions.length})</h3>
                {pendingActions.map((action: ActionItem) => (
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
                            {action.assignee?.displayName || 'Assigné'}
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
              </div>
            )}

            {/* Completed actions */}
            {completedActions.length > 0 && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium text-gray-500">Terminées ({completedActions.length})</h3>
                {completedActions.map((action: ActionItem) => (
                  <div
                    key={action.id}
                    className="flex items-center gap-3 p-2 bg-gray-50 rounded text-sm text-gray-500"
                  >
                    <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
                    <span className="line-through">{action.title}</span>
                  </div>
                ))}
              </div>
            )}

            {actions.length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">
                Aucune action créée
              </p>
            )}
          </div>
        </div>

        {/* Template info */}
        {template && (
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h3 className="text-sm font-medium text-gray-500 mb-3">Template utilisé</h3>
            <div className="flex items-center gap-4">
              <span className="font-medium text-gray-900">{template.name}</span>
              <div className="flex gap-2">
                {template.columns.map((column) => (
                  <div
                    key={column.id}
                    className="flex items-center gap-1 px-2 py-1 rounded text-xs"
                    style={{ backgroundColor: column.color + '20' }}
                  >
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: column.color }}
                    />
                    <span>{column.name}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>
    )
  }

  // Default view for draft/active retros
  return (
    <div className="space-y-6">
      <button
        onClick={() => navigate(`/teams/${teamId}`)}
        className="flex items-center gap-2 text-gray-600 hover:text-gray-900"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to team
      </button>

      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{retro.name}</h1>
            <p className="text-gray-500 mt-1">
              Template: {template?.name || 'Loading...'} | Status: <strong>{retro.status}</strong>
            </p>
          </div>
          <div className="flex items-center gap-3">
            {/* Always show Join button for active retros */}
            {retro.status === 'active' && (
              <button
                onClick={() => navigate(`/retro/${retroId}`)}
                className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                <Play className="w-4 h-4" />
                Rejoindre la Retro
              </button>
            )}
            {retro.status === 'draft' && (
              <>
                <button
                  onClick={() => startMutation.mutate()}
                  disabled={startMutation.isPending}
                  className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
                >
                  <Play className="w-4 h-4" />
                  {startMutation.isPending ? 'Démarrage...' : 'Démarrer la Retro'}
                </button>
                <button
                  onClick={() => {
                    if (confirm('Êtes-vous sûr de vouloir supprimer cette rétrospective ?')) {
                      deleteMutation.mutate()
                    }
                  }}
                  className="p-2 text-red-600 hover:bg-red-50 rounded-lg"
                >
                  <Trash2 className="w-5 h-5" />
                </button>
              </>
            )}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-6">
          <div>
            <h3 className="text-sm font-medium text-gray-500">Status</h3>
            <p className="mt-1 text-gray-900 capitalize">{retro.status}</p>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-500">Current Phase</h3>
            <p className="mt-1 text-gray-900 capitalize">{retro.currentPhase}</p>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-500">Max Votes per User</h3>
            <p className="mt-1 text-gray-900">{retro.maxVotesPerUser}</p>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-500">Anonymous Voting</h3>
            <p className="mt-1 text-gray-900">{retro.anonymousVoting ? 'Yes' : 'No'}</p>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-500">Created</h3>
            <p className="mt-1 text-gray-900">
              {new Date(retro.createdAt).toLocaleString()}
            </p>
          </div>
          {retro.startedAt && (
            <div>
              <h3 className="text-sm font-medium text-gray-500">Started</h3>
              <p className="mt-1 text-gray-900">
                {new Date(retro.startedAt).toLocaleString()}
              </p>
            </div>
          )}
        </div>

        {template && (
          <div className="mt-6 pt-6 border-t border-gray-200">
            <h3 className="text-sm font-medium text-gray-500 mb-3">Columns</h3>
            <div className="flex gap-3">
              {template.columns.map((column) => (
                <div
                  key={column.id}
                  className="flex items-center gap-2 px-3 py-2 rounded-lg"
                  style={{ backgroundColor: column.color + '20' }}
                >
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: column.color }}
                  />
                  <span className="text-sm font-medium">{column.name}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
