import { useState, useRef, useEffect, useCallback } from 'react'
import { DndContext, DragEndEvent, DragOverlay, DragStartEvent, pointerWithin } from '@dnd-kit/core'
import { useRetroStore } from '../../store/retroStore'
import { useAuthStore } from '../../store/authStore'
import ItemCard from './ItemCard'
import DraftCard from './DraftCard'
import type { Template, RetroPhase, Item } from '../../types'
import { Plus } from 'lucide-react'

interface RetroBoardProps {
  template: Template
  currentPhase: RetroPhase
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

export default function RetroBoard({
  template,
  currentPhase,
  isFacilitator,
  send,
}: RetroBoardProps) {
  const { items, participants, drafts, retro, myVotesOnItems } = useRetroStore()
  const { user } = useAuthStore()
  const [newItemContent, setNewItemContent] = useState<Record<string, string>>({})
  const [activeItem, setActiveItem] = useState<Item | null>(null)
  const typingTimeoutRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({})

  const canAddItems = currentPhase === 'brainstorm'
  const canVote = currentPhase === 'vote'
  const canGroup = currentPhase === 'group' && isFacilitator

  // Compute total votes used by current user (for multi-vote limits)
  const myTotalVotes = Array.from(myVotesOnItems.values()).reduce((sum, count) => sum + count, 0)
  const maxVotesPerUser = retro?.maxVotesPerUser ?? 5
  const maxVotesPerItem = retro?.maxVotesPerItem ?? 3

  // Broadcast typing status with debounce
  const broadcastTyping = useCallback((columnId: string, content: string) => {
    // Clear previous timeout for this column
    if (typingTimeoutRef.current[columnId]) {
      clearTimeout(typingTimeoutRef.current[columnId])
    }

    if (content.length > 0) {
      // Send typing status
      send('draft_typing', { columnId, contentLength: content.length })

      // Set timeout to clear draft after 3 seconds of inactivity
      typingTimeoutRef.current[columnId] = setTimeout(() => {
        send('draft_clear', { columnId })
      }, 3000)
    } else {
      // Clear draft immediately if content is empty
      send('draft_clear', { columnId })
    }
  }, [send])

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      Object.values(typingTimeoutRef.current).forEach(clearTimeout)
    }
  }, [])

  // Get drafts for a specific column (excluding current user's drafts)
  const getColumnDrafts = (columnId: string) => {
    const columnDrafts: Array<{ userId: string; userName: string; contentLength: number }> = []
    drafts.forEach((draft) => {
      if (draft.columnId === columnId && draft.userId !== user?.id) {
        columnDrafts.push({
          userId: draft.userId,
          userName: draft.userName,
          contentLength: draft.contentLength,
        })
      }
    })
    return columnDrafts
  }

  // Helper to get column name from columnId
  const getColumnName = (columnId: string): string => {
    const column = template.columns.find(c => c.id === columnId)
    return column?.name || columnId
  }

  // Helper to get author name from authorId
  // Returns a loading indicator if participants list is empty but items exist (reload in progress)
  const getAuthorName = (authorId: string): string => {
    const participant = participants.find(p => p.userId === authorId)
    if (participant) {
      return participant.name
    }
    // If no participants but we have items, we're likely in a loading state (page reload)
    if (participants.length === 0 && items.length > 0) {
      return '...'
    }
    // Check if the author is the current user (might not be in participants list yet)
    if (authorId === user?.id && user?.displayName) {
      return user.displayName
    }
    return 'Inconnu'
  }

  // Get items that are children of a parent (grouped under it)
  const getGroupedItems = (parentId: string): Item[] => {
    return items.filter((item) => item.groupId === parentId)
  }

  const getColumnItems = (columnId: string): Item[] => {
    return items
      .filter((item) => item.columnId === columnId && !item.groupId)
      .sort((a, b) => {
        // Sort by vote count in vote/discuss phase
        if (currentPhase === 'vote' || currentPhase === 'discuss') {
          return b.voteCount - a.voteCount
        }
        return a.position - b.position
      })
  }

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event
    const draggedItem = items.find((item) => item.id === active.id)
    setActiveItem(draggedItem || null)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveItem(null)

    if (!over || !canGroup) return

    // Extract the target item ID from the droppable ID
    const targetId = over.id.toString().replace('drop-', '')
    const sourceId = active.id.toString()

    // Don't group item with itself
    if (sourceId === targetId) return

    // Send WebSocket message to group items
    send('item_group', {
      parentId: targetId,
      childIds: [sourceId],
    })
  }

  const handleAddItem = (columnId: string) => {
    const content = newItemContent[columnId]?.trim()
    if (!content) return

    // Clear any pending typing timeout
    if (typingTimeoutRef.current[columnId]) {
      clearTimeout(typingTimeoutRef.current[columnId])
    }

    // Clear draft and create item
    send('draft_clear', { columnId })
    send('item_create', { columnId, content })
    setNewItemContent((prev) => ({ ...prev, [columnId]: '' }))
  }

  const handleInputChange = (columnId: string, value: string) => {
    setNewItemContent((prev) => ({ ...prev, [columnId]: value }))
    broadcastTyping(columnId, value)
  }

  const handleVote = (itemId: string) => {
    send('vote_add', { itemId })
  }

  const handleUnvote = (itemId: string) => {
    send('vote_remove', { itemId })
  }

  const handleUpdateItem = (itemId: string, content: string) => {
    send('item_update', { itemId, content })
  }

  const handleDeleteItem = (itemId: string) => {
    send('item_delete', { itemId })
  }

  return (
    <DndContext
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      collisionDetection={pointerWithin}
    >
      <div className="flex gap-4 h-full">
        {template.columns.map((column) => (
          <div
            key={column.id}
            className="flex-1 min-w-[280px] max-w-[400px] bg-gray-50 rounded-lg p-4"
          >
            {/* Column Header */}
            <div
              className="flex items-center gap-2 mb-4 pb-2 border-b-2"
              style={{ borderColor: column.color }}
            >
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: column.color }}
              />
              <h3 className="font-semibold text-gray-900">{column.name}</h3>
              <span className="ml-auto text-sm text-gray-500">
                {getColumnItems(column.id).length}
              </span>
              {canGroup && (
                <span className="text-xs text-primary-600 bg-primary-50 px-2 py-0.5 rounded">
                  Glisser pour grouper
                </span>
              )}
            </div>

            {/* Items */}
            <div className="space-y-3 mb-4 max-h-[calc(100vh-300px)] overflow-y-auto">
              {/* Other users' drafts (shown as masked content) */}
              {canAddItems && getColumnDrafts(column.id).map((draft) => (
                <DraftCard
                  key={`draft-${draft.userId}`}
                  userName={draft.userName}
                  contentLength={draft.contentLength}
                  color={column.color}
                />
              ))}
              {getColumnItems(column.id).map((item) => {
                // Obfuscate other users' items during brainstorm phase
                const isObfuscated = currentPhase === 'brainstorm' && item.authorId !== user?.id
                const myVoteCountOnItem = myVotesOnItems.get(item.id) || 0
                const canAddVoteOnItem = myTotalVotes < maxVotesPerUser && myVoteCountOnItem < maxVotesPerItem
                return (
                  <ItemCard
                    key={item.id}
                    item={item}
                    canVote={canVote}
                    canEdit={item.authorId === user?.id && currentPhase === 'brainstorm'}
                    canDelete={item.authorId === user?.id && currentPhase === 'brainstorm'}
                    canGroup={canGroup}
                    isObfuscated={isObfuscated}
                    groupedItems={getGroupedItems(item.id)}
                    columnName={column.name}
                    authorName={getAuthorName(item.authorId)}
                    myVoteCount={myVoteCountOnItem}
                    canAddVote={canAddVoteOnItem}
                    getColumnName={getColumnName}
                    getAuthorName={getAuthorName}
                    onVote={() => handleVote(item.id)}
                    onUnvote={() => handleUnvote(item.id)}
                    onUpdate={(content) => handleUpdateItem(item.id, content)}
                    onDelete={() => handleDeleteItem(item.id)}
                    color={column.color}
                  />
                )
              })}
            </div>

          {/* Add Item */}
          {canAddItems && (
            <div className="mt-auto">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newItemContent[column.id] || ''}
                  onChange={(e) => handleInputChange(column.id, e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleAddItem(column.id)
                    }
                  }}
                  placeholder="Ajouter un élément..."
                  className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <button
                  onClick={() => handleAddItem(column.id)}
                  disabled={!newItemContent[column.id]?.trim()}
                  className="p-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Plus className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>
      ))}
      </div>

      {/* Drag Overlay - shows preview of dragged item */}
      <DragOverlay>
        {activeItem && (
          <div className="bg-white rounded-lg border border-gray-200 p-3 shadow-lg opacity-90">
            <p className="text-sm text-gray-800">{activeItem.content}</p>
          </div>
        )}
      </DragOverlay>
    </DndContext>
  )
}
