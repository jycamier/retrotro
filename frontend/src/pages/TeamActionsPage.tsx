import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { teamsApi } from '../api/client'
import { 
  CheckCircle2, 
  Circle, 
  Clock, 
  Calendar, 
  Users, 
  LayoutGrid,
  ArrowUp,
  Filter
} from 'lucide-react'
import type { ActionItem } from '../types'

type KanbanColumn = 'todo' | 'in_progress' | 'done'

interface Column {
  id: KanbanColumn
  title: string
  color: string
}

const COLUMNS: Column[] = [
  { id: 'todo', title: 'À faire', color: '#6B7280' },
  { id: 'in_progress', title: 'En cours', color: '#F59E0B' },
  { id: 'done', title: 'Terminé', color: '#10B981' },
]

function getActionsByStatus(actions: ActionItem[], status: KanbanColumn): ActionItem[] {
  return actions.filter(action => {
    if (status === 'done') return action.isCompleted
    if (status === 'todo') return !action.isCompleted && !action.dueDate
    if (status === 'in_progress') return !action.isCompleted && !!action.dueDate
    return false
  })
}

function formatDueDate(dateString?: string): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('fr-FR', { 
    day: 'numeric', 
    month: 'short',
    year: 'numeric'
  })
}

function isOverdue(dueDate?: string): boolean {
  if (!dueDate) return false
  return new Date(dueDate) < new Date()
}

export default function TeamActionsPage() {
  const { teamId } = useParams<{ teamId: string }>()
  const [filterRetro, setFilterRetro] = useState<string>('all')

  const { data: team, isLoading: teamLoading } = useQuery({
    queryKey: ['team', teamId],
    queryFn: () => teamsApi.get(teamId!),
    enabled: !!teamId,
  })

  const { data: actions, isLoading: actionsLoading } = useQuery({
    queryKey: ['teamActions', teamId],
    queryFn: () => teamsApi.getActions(teamId!),
    enabled: !!teamId,
  })

  const retroNames = Array.from(new Set((actions || []).map(a => a.retroName).filter(Boolean)))

  const filteredActions = filterRetro === 'all' 
    ? actions 
    : actions?.filter(a => a.retroName === filterRetro)

  if (teamLoading || actionsLoading) {
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
      {/* Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to={`/teams/${teamId}`}
              className="p-3 bg-primary-100 rounded-lg hover:bg-primary-200 transition-colors"
            >
              <Users className="w-6 h-6 text-primary-600" />
            </Link>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Actions de l'équipe</h1>
              <p className="text-gray-500">@{team.name}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-gray-500" />
              <select
                value={filterRetro}
                onChange={(e) => setFilterRetro(e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
              >
                <option value="all">Tous les rétro</option>
                {retroNames.map(name => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="mt-6 grid grid-cols-3 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-gray-600">
              <Circle className="w-4 h-4" />
              <span className="text-sm">À faire</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">
              {getActionsByStatus(actions || [], 'todo').length}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-amber-600">
              <Clock className="w-4 h-4" />
              <span className="text-sm">En cours</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">
              {getActionsByStatus(actions || [], 'in_progress').length}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-green-600">
              <CheckCircle2 className="w-4 h-4" />
              <span className="text-sm">Terminé</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">
              {getActionsByStatus(actions || [], 'done').length}
            </p>
          </div>
        </div>
      </div>

      {/* Kanban Board */}
      <div className="grid grid-cols-3 gap-6">
        {COLUMNS.map(column => {
          const columnActions = getActionsByStatus(filteredActions || [], column.id)
          
          return (
            <div key={column.id} className="bg-gray-100 rounded-lg p-4">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <div 
                    className="w-3 h-3 rounded-full" 
                    style={{ backgroundColor: column.color }}
                  />
                  <h2 className="font-semibold text-gray-900">{column.title}</h2>
                </div>
                <span className="text-sm text-gray-500 bg-white px-2 py-1 rounded-full">
                  {columnActions.length}
                </span>
              </div>

              <div className="space-y-3">
                {columnActions.length === 0 ? (
                  <div className="text-center py-8 text-gray-400 text-sm">
                    Aucune action
                  </div>
                ) : (
                  columnActions.map(action => (
                    <ActionCard 
                      key={action.id} 
                      action={action} 
                    />
                  ))
                )}
              </div>
            </div>
          )
        })}
      </div>

      {(!actions || actions.length === 0) && (
        <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
          <LayoutGrid className="w-12 h-12 text-gray-400 mx-auto" />
          <h3 className="mt-4 text-lg font-medium text-gray-900">Aucune action</h3>
          <p className="mt-2 text-gray-600">
            Les actions créées dans les rétrospectives terminées apparaîtront ici
          </p>
          <Link
            to={`/teams/${teamId}`}
            className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
          >
            Voir les rétrospectives
          </Link>
        </div>
      )}
    </div>
  )
}

function ActionCard({ action }: { action: ActionItem }) {
  const [isCompleted, setIsCompleted] = useState(action.isCompleted)
  const overdue = isOverdue(action.dueDate) && !isCompleted

  const handleToggle = () => {
    setIsCompleted(!isCompleted)
    // In a real app, this would call the API
    console.log('Toggle action:', action.id, !isCompleted)
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm hover:shadow-md transition-shadow">
      {/* Retro name */}
      {action.retroName && (
        <div className="text-xs text-gray-500 mb-2">
          <Calendar className="w-3 h-3 inline mr-1" />
          {action.retroName}
        </div>
      )}

      {/* Title */}
      <h3 className={`font-medium text-gray-900 ${isCompleted ? 'line-through text-gray-400' : ''}`}>
        {action.title}
      </h3>

      {/* Description */}
      {action.description && (
        <p className="text-sm text-gray-600 mt-2 line-clamp-2">
          {action.description}
        </p>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-gray-100">
        <button
          onClick={handleToggle}
          className={`flex items-center gap-1.5 text-sm transition-colors ${
            isCompleted 
              ? 'text-green-600 hover:text-green-700' 
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          {isCompleted ? (
            <CheckCircle2 className="w-4 h-4" />
          ) : (
            <Circle className="w-4 h-4" />
          )}
          {isCompleted ? 'Terminé' : 'Marquer terminé'}
        </button>

        {action.dueDate && (
          <div className={`flex items-center gap-1 text-xs ${
            overdue ? 'text-red-600' : 'text-gray-500'
          }`}>
            <Clock className="w-3 h-3" />
            {formatDueDate(action.dueDate)}
          </div>
        )}
      </div>

      {/* Priority indicator */}
      {action.priority > 0 && (
        <div className="flex items-center gap-1 mt-3 text-xs text-gray-500">
          <ArrowUp className="w-3 h-3" />
          Priorité {action.priority}
        </div>
      )}
    </div>
  )
}
