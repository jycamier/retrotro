import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import { teamsApi, retrosApi, templatesApi } from '../api/client'
import { Plus, Play, Calendar, CheckCircle, Clock, Users, Star, BarChart3, LayoutGrid, Coffee, MessageSquare } from 'lucide-react'
import type { Retrospective, Template, RotiResults, SessionType } from '../types'

export default function TeamPage() {
  const { teamId } = useParams<{ teamId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newRetroName, setNewRetroName] = useState('')
  const [selectedTemplateId, setSelectedTemplateId] = useState('')
  const [sessionType, setSessionType] = useState<SessionType>('retro')
  const [lcTopicTimebox, setLcTopicTimebox] = useState(5) // minutes

  const openCreateModal = () => {
    const today = new Date().toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
    setNewRetroName(`Rétro du ${today}`)
    setSessionType('retro')
    setSelectedTemplateId('')
    setLcTopicTimebox(5)
    setShowCreateModal(true)
  }

  const { data: team, isLoading: teamLoading } = useQuery({
    queryKey: ['team', teamId],
    queryFn: () => teamsApi.get(teamId!),
    enabled: !!teamId,
  })

  const { data: retros, isLoading: retrosLoading } = useQuery({
    queryKey: ['retros', teamId],
    queryFn: () => retrosApi.list(teamId!),
    enabled: !!teamId,
  })

  const { data: templates } = useQuery({
    queryKey: ['templates', teamId],
    queryFn: () => templatesApi.list(teamId),
    enabled: !!teamId,
  })

  // Fetch ROTI results for completed retros
  const completedRetroIds = (retros || [])
    .filter((r: Retrospective) => r.status === 'completed')
    .map((r: Retrospective) => r.id)

  const rotiQueries = useQueries({
    queries: completedRetroIds.map((retroId: string) => ({
      queryKey: ['roti', retroId],
      queryFn: () => retrosApi.getRotiResults(retroId),
      enabled: !!retroId,
      staleTime: 1000 * 60 * 5, // Cache for 5 minutes
    })),
  })

  // Create a map of retroId -> rotiResults
  const rotiResultsMap = new Map<string, RotiResults>()
  completedRetroIds.forEach((retroId: string, index: number) => {
    const query = rotiQueries[index]
    if (query?.data) {
      rotiResultsMap.set(retroId, query.data)
    }
  })

  const createRetroMutation = useMutation({
    mutationFn: (data: { name: string; teamId: string; templateId?: string; sessionType?: SessionType; lcTopicTimeboxSeconds?: number }) =>
      retrosApi.create(data),
    onSuccess: (retro: Retrospective) => {
      queryClient.invalidateQueries({ queryKey: ['retros', teamId] })
      setShowCreateModal(false)
      setNewRetroName('')
      setSelectedTemplateId('')
      // Auto-start and navigate to the session for LC
      if (retro.sessionType === 'lean_coffee') {
        retrosApi.start(retro.id).then(() => {
          navigate(`/leancoffee/${retro.id}`)
        })
      }
    },
  })

  const handleCreateRetro = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRetroName || !teamId) return

    if (sessionType === 'lean_coffee') {
      createRetroMutation.mutate({
        name: newRetroName,
        teamId,
        sessionType: 'lean_coffee',
        lcTopicTimeboxSeconds: lcTopicTimebox * 60,
      })
    } else {
      if (!selectedTemplateId) return
      createRetroMutation.mutate({
        name: newRetroName,
        teamId,
        templateId: selectedTemplateId,
        sessionType: 'retro',
      })
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'draft':
        return <Clock className="w-4 h-4 text-gray-500" />
      case 'active':
        return <Play className="w-4 h-4 text-green-500" />
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-blue-500" />
      default:
        return null
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'draft':
        return 'Draft'
      case 'active':
        return 'In Progress'
      case 'completed':
        return 'Completed'
      case 'archived':
        return 'Archived'
      default:
        return status
    }
  }

  if (teamLoading || retrosLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  if (!team) {
    return <div>Team not found</div>
  }

  return (
    <div className="space-y-6">
      {/* Team Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary-100 rounded-lg">
              <Users className="w-6 h-6 text-primary-600" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{team.name}</h1>
              <p className="text-gray-500">@{team.slug}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Link
              to={`/teams/${teamId}/actions`}
              className="flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <LayoutGrid className="w-4 h-4" />
              Actions
            </Link>
            <Link
              to={`/teams/${teamId}/stats`}
              className="flex items-center gap-2 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <BarChart3 className="w-4 h-4" />
              Statistiques
            </Link>
            <button
              onClick={openCreateModal}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
            >
              <Plus className="w-4 h-4" />
              Nouvelle session
            </button>
          </div>
        </div>
        {team.description && (
          <p className="mt-4 text-gray-600">{team.description}</p>
        )}
      </div>

      {/* Retrospectives List */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Retrospectives</h2>

        {retros && retros.length > 0 ? (
          <div className="space-y-3">
            {retros.map((retro: Retrospective) => (
              <Link
                key={retro.id}
                to={retro.status === 'active'
                  ? (retro.sessionType === 'lean_coffee' ? `/leancoffee/${retro.id}` : `/retro/${retro.id}`)
                  : `/teams/${teamId}/retros/${retro.id}`}
                className="block bg-white rounded-lg border border-gray-200 p-4 hover:border-primary-300 hover:shadow-md transition-all"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {getStatusIcon(retro.status)}
                    <div>
                      <h3 className="font-medium text-gray-900">{retro.name}</h3>
                      <p className="text-sm text-gray-500">
                        {getStatusLabel(retro.status)} · Created{' '}
                        {new Date(retro.createdAt).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {retro.status === 'completed' && rotiResultsMap.has(retro.id) && (
                      <div className="flex items-center gap-1 px-2 py-1 bg-yellow-50 rounded text-yellow-700">
                        <Star className="w-3.5 h-3.5 fill-yellow-400 text-yellow-400" />
                        <span className="text-xs font-medium">
                          {rotiResultsMap.get(retro.id)!.average.toFixed(1)}
                        </span>
                      </div>
                    )}
                    <span className={`px-2 py-1 text-xs rounded ${
                      retro.status === 'active'
                        ? 'bg-green-100 text-green-700'
                        : retro.status === 'completed'
                        ? 'bg-blue-100 text-blue-700'
                        : 'bg-gray-100 text-gray-700'
                    }`}>
                      {getStatusLabel(retro.status)}
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
            <Calendar className="w-12 h-12 text-gray-400 mx-auto" />
            <h3 className="mt-4 text-lg font-medium text-gray-900">No retrospectives yet</h3>
            <p className="mt-2 text-gray-600">
              Create your first retrospective to get started
            </p>
            <button
              onClick={openCreateModal}
              className="mt-4 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              Create retrospective
            </button>
          </div>
        )}
      </div>

      {/* Create Session Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-lg">
            <h2 className="text-lg font-semibold mb-4">Nouvelle session</h2>
            <form onSubmit={handleCreateRetro} className="space-y-4">
              {/* Session Type Selector */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Type de session
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => {
                      setSessionType('retro')
                      const today = new Date().toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
                      setNewRetroName(`Rétro du ${today}`)
                    }}
                    className={`flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-colors ${
                      sessionType === 'retro'
                        ? 'border-primary-500 bg-primary-50 text-primary-700'
                        : 'border-gray-200 hover:border-gray-300 text-gray-600'
                    }`}
                  >
                    <MessageSquare className="w-6 h-6" />
                    <span className="font-medium text-sm">Rétrospective</span>
                    <span className="text-xs text-center opacity-75">Brainstorm, grouper, voter, discuter</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setSessionType('lean_coffee')
                      const today = new Date().toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
                      setNewRetroName(`Lean Coffee du ${today}`)
                      setSelectedTemplateId('')
                    }}
                    className={`flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-colors ${
                      sessionType === 'lean_coffee'
                        ? 'border-amber-500 bg-amber-50 text-amber-700'
                        : 'border-gray-200 hover:border-gray-300 text-gray-600'
                    }`}
                  >
                    <Coffee className="w-6 h-6" />
                    <span className="font-medium text-sm">Lean Coffee</span>
                    <span className="text-xs text-center opacity-75">Proposer, voter, discuter avec timebox</span>
                  </button>
                </div>
              </div>

              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Nom
                </label>
                <input
                  type="text"
                  value={newRetroName}
                  onChange={(e) => setNewRetroName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  placeholder={sessionType === 'lean_coffee' ? 'Lean Coffee du 25 février' : 'Sprint 42 Retrospective'}
                  required
                />
              </div>

              {/* Template (retro only) */}
              {sessionType === 'retro' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Template
                  </label>
                  <select
                    value={selectedTemplateId}
                    onChange={(e) => setSelectedTemplateId(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    required
                  >
                    <option value="">Choisir un template...</option>
                    {templates?.map((template: Template) => (
                      <option key={template.id} value={template.id}>
                        {template.name} {template.isBuiltIn && '(Built-in)'}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Timebox (LC only) */}
              {sessionType === 'lean_coffee' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Timebox par sujet (minutes)
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={30}
                    value={lcTopicTimebox}
                    onChange={(e) => setLcTopicTimebox(Number(e.target.value))}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Durée par défaut pour chaque sujet. Extensible avec le bouton +5 min.
                  </p>
                </div>
              )}

              <div className="flex justify-end gap-3 mt-6">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg"
                >
                  Annuler
                </button>
                <button
                  type="submit"
                  disabled={createRetroMutation.isPending}
                  className={`px-4 py-2 text-white rounded-lg disabled:opacity-50 ${
                    sessionType === 'lean_coffee'
                      ? 'bg-amber-600 hover:bg-amber-700'
                      : 'bg-primary-600 hover:bg-primary-700'
                  }`}
                >
                  {createRetroMutation.isPending ? 'Création...' : 'Créer'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
