import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { teamsApi } from '../api/client'
import {
  DndContext,
  DragOverlay,
  useDroppable,
  useDraggable,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import {
  CheckCircle2,
  Circle,
  Clock,
  Calendar,
  Users,
  LayoutGrid,
  ArrowUp,
  Filter,
  UserCircle,
  ChevronDown,
  MessageSquare,
  StickyNote,
} from 'lucide-react'
import type { ActionItem, TeamMember } from '../types'

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
  return actions.filter(action => action.status === status)
}

function formatDueDate(dateString?: string): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('fr-FR', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function isOverdue(dueDate?: string): boolean {
  if (!dueDate) return false
  return new Date(dueDate) < new Date()
}

// --- Droppable Column ---
function DroppableColumn({
  column,
  children,
}: {
  column: Column
  children: React.ReactNode
}) {
  const { setNodeRef, isOver } = useDroppable({ id: column.id })

  return (
    <div
      ref={setNodeRef}
      data-testid={`column-${column.id}`}
      className={`rounded-lg p-4 transition-colors ${
        isOver ? 'bg-primary-50 ring-2 ring-primary-300' : 'bg-gray-100'
      }`}
    >
      {children}
    </div>
  )
}

// --- Draggable Action Card Wrapper ---
function DraggableActionCard({
  action,
  members,
  patchMutation,
}: {
  action: ActionItem
  members: TeamMember[]
  patchMutation: ReturnType<typeof useMutation<ActionItem, Error, { actionId: string; data: { status?: string; assigneeId?: string | null; description?: string } }, { previous: ActionItem[] | undefined }>>
}) {
  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: action.id,
  })

  const style = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
      }
    : undefined

  return (
    <div ref={setNodeRef} style={style} {...listeners} {...attributes}>
      <ActionCard action={action} members={members} patchMutation={patchMutation} />
    </div>
  )
}

// --- Action Card ---
function ActionCard({
  action,
  members,
  patchMutation,
  isOverlay,
}: {
  action: ActionItem
  members: TeamMember[]
  patchMutation: ReturnType<typeof useMutation<ActionItem, Error, { actionId: string; data: { status?: string; assigneeId?: string | null; description?: string } }, { previous: ActionItem[] | undefined }>>
  isOverlay?: boolean
}) {
  const [showDropdown, setShowDropdown] = useState(false)
  const [showNotes, setShowNotes] = useState(false)
  const [notesDraft, setNotesDraft] = useState(action.description || '')
  const dropdownRef = useRef<HTMLDivElement>(null)
  const overdue = isOverdue(action.dueDate) && action.status !== 'done'

  // Close dropdown on click outside
  useEffect(() => {
    if (!showDropdown) return
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showDropdown])

  // Sync draft when action.description changes externally
  useEffect(() => {
    if (!showNotes) {
      setNotesDraft(action.description || '')
    }
  }, [action.description, showNotes])

  const assignee = members.find(m => m.userId === action.assigneeId)
  const assigneeLabel = assignee
    ? assignee.user?.displayName
      ? assignee.user.displayName
          .split(' ')
          .map(p => p[0])
          .join('')
          .toUpperCase()
          .slice(0, 2)
      : assignee.userId.slice(0, 2).toUpperCase()
    : null

  const handleSelectAssignee = (userId: string | null) => {
    setShowDropdown(false)
    patchMutation.mutate({ actionId: action.id, data: { assigneeId: userId } })
  }

  const handleSaveNotes = () => {
    const trimmed = notesDraft.trim()
    if (trimmed !== (action.description || '')) {
      patchMutation.mutate({ actionId: action.id, data: { description: trimmed || '' } })
    }
    setShowNotes(false)
  }

  const handleCardClick = () => {
    if (!isOverlay) {
      setShowNotes(!showNotes)
    }
  }

  return (
    <div
      className={`bg-white rounded-lg border border-gray-200 p-4 shadow-sm ${
        isOverlay ? 'shadow-lg rotate-2 opacity-90' : 'hover:shadow-md cursor-pointer'
      } transition-shadow`}
      onClick={handleCardClick}
    >
      {/* Retro name */}
      {action.retroName && (
        <div className="text-xs text-gray-500 mb-2">
          <Calendar className="w-3 h-3 inline mr-1" />
          {action.retroName}
        </div>
      )}

      {/* Source item */}
      {action.itemContent && (
        <div className="text-xs text-gray-400 mb-2 flex items-start gap-1">
          <MessageSquare className="w-3 h-3 mt-0.5 flex-shrink-0" />
          <span className="line-clamp-2 italic">{action.itemContent}</span>
        </div>
      )}

      {/* Title */}
      <div className="flex items-center justify-between">
        <h3
          className={`font-medium text-gray-900 ${
            action.status === 'done' ? 'line-through text-gray-400' : ''
          }`}
        >
          {action.title}
        </h3>
        <StickyNote className={`w-4 h-4 flex-shrink-0 ${action.description ? 'text-amber-500' : 'text-gray-300'}`} />
      </div>

      {/* Description preview (when collapsed) */}
      {!showNotes && action.description && (
        <p className="text-sm text-gray-600 mt-2 line-clamp-2">{action.description}</p>
      )}

      {/* Notes editor (when expanded) */}
      {showNotes && (
        <div
          className="mt-3"
          onClick={(e) => e.stopPropagation()}
          onPointerDown={(e) => e.stopPropagation()}
        >
          <textarea
            value={notesDraft}
            onChange={(e) => setNotesDraft(e.target.value)}
            placeholder="Ajouter des notes..."
            rows={3}
            className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
            autoFocus
          />
          <div className="flex justify-end gap-2 mt-2">
            <button
              onClick={() => { setNotesDraft(action.description || ''); setShowNotes(false) }}
              className="px-3 py-1 text-xs text-gray-600 hover:text-gray-800 border border-gray-300 rounded-lg"
            >
              Annuler
            </button>
            <button
              onClick={handleSaveNotes}
              className="px-3 py-1 text-xs bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              Enregistrer
            </button>
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-gray-100">
        {/* Assignee dropdown */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={(e) => {
              e.stopPropagation()
              setShowDropdown(!showDropdown)
            }}
            onPointerDown={(e) => e.stopPropagation()}
            className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 transition-colors"
          >
            {assigneeLabel ? (
              <span className="w-6 h-6 rounded-full bg-primary-100 text-primary-700 flex items-center justify-center text-xs font-medium">
                {assigneeLabel}
              </span>
            ) : (
              <UserCircle className="w-5 h-5" />
            )}
            <span className="text-xs">
              {assignee
                ? assignee.user?.displayName || assignee.userId
                : 'Non assigné'}
            </span>
            <ChevronDown className="w-3 h-3" />
          </button>

          {showDropdown && (
            <div className="absolute z-50 mt-1 left-0 w-52 bg-white border border-gray-200 rounded-lg shadow-lg py-1">
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  handleSelectAssignee(null)
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="w-full text-left px-3 py-2 text-sm text-gray-500 hover:bg-gray-50"
              >
                Non assigné
              </button>
              {members.map(member => (
                <button
                  key={member.id}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleSelectAssignee(member.userId)
                  }}
                  onPointerDown={(e) => e.stopPropagation()}
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 ${
                    member.userId === action.assigneeId
                      ? 'bg-primary-50 text-primary-700 font-medium'
                      : 'text-gray-700'
                  }`}
                >
                  {member.user?.displayName || member.userId}
                </button>
              ))}
            </div>
          )}
        </div>

        {action.dueDate && (
          <div
            className={`flex items-center gap-1 text-xs ${
              overdue ? 'text-red-600' : 'text-gray-500'
            }`}
          >
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

// --- Main Page ---
export default function TeamActionsPage() {
  const { teamId } = useParams<{ teamId: string }>()
  const queryClient = useQueryClient()
  const [filterRetro, setFilterRetro] = useState<string>('all')
  const [activeId, setActiveId] = useState<string | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  )

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

  const { data: members } = useQuery({
    queryKey: ['teamMembers', teamId],
    queryFn: () => teamsApi.getMembers(teamId!),
    enabled: !!teamId,
  })

  const patchMutation = useMutation<
    ActionItem,
    Error,
    { actionId: string; data: { status?: string; assigneeId?: string | null; description?: string } },
    { previous: ActionItem[] | undefined }
  >({
    mutationFn: ({ actionId, data }) => teamsApi.patchAction(teamId!, actionId, data),
    onMutate: async ({ actionId, data }) => {
      await queryClient.cancelQueries({ queryKey: ['teamActions', teamId] })
      const previous = queryClient.getQueryData<ActionItem[]>(['teamActions', teamId])
      queryClient.setQueryData<ActionItem[]>(['teamActions', teamId], old =>
        old?.map(a => (a.id === actionId ? { ...a, ...data, assigneeId: data.assigneeId ?? undefined } as ActionItem : a)) ?? []
      )
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['teamActions', teamId], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['teamActions', teamId] })
    },
  })

  const retroNames = Array.from(
    new Set((actions || []).map(a => a.retroName).filter(Boolean))
  )

  const filteredActions =
    filterRetro === 'all'
      ? actions
      : actions?.filter(a => a.retroName === filterRetro)

  const activeAction = activeId
    ? (filteredActions || []).find(a => a.id === activeId) ?? null
    : null

  function handleDragStart(event: DragStartEvent) {
    setActiveId(event.active.id as string)
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveId(null)
    const { active, over } = event
    if (!over) return

    const actionId = active.id as string
    const newStatus = over.id as KanbanColumn

    // Find the current action to check its status
    const action = (actions || []).find(a => a.id === actionId)
    if (!action || action.status === newStatus) return

    patchMutation.mutate({ actionId, data: { status: newStatus } })
  }

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
                onChange={e => setFilterRetro(e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
              >
                <option value="all">Tous les rétro</option>
                {retroNames.map(name => (
                  <option key={name} value={name}>
                    {name}
                  </option>
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

      {/* Kanban Board with DnD */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="grid grid-cols-3 gap-6">
          {COLUMNS.map(column => {
            const columnActions = getActionsByStatus(filteredActions || [], column.id)

            return (
              <DroppableColumn key={column.id} column={column}>
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
                      <DraggableActionCard
                        key={action.id}
                        action={action}
                        members={members || []}
                        patchMutation={patchMutation}
                      />
                    ))
                  )}
                </div>
              </DroppableColumn>
            )
          })}
        </div>

        <DragOverlay>
          {activeAction ? (
            <ActionCard
              action={activeAction}
              members={members || []}
              patchMutation={patchMutation}
              isOverlay
            />
          ) : null}
        </DragOverlay>
      </DndContext>

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
