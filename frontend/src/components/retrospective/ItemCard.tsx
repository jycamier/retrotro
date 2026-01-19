import { useState } from 'react'
import { ThumbsUp, Edit2, Trash2, Check, X, Layers, GripVertical } from 'lucide-react'
import { useDraggable, useDroppable } from '@dnd-kit/core'
import type { Item } from '../../types'
import clsx from 'clsx'

interface ItemCardProps {
  item: Item
  canVote: boolean
  canEdit: boolean
  canDelete: boolean
  canGroup: boolean
  isObfuscated?: boolean
  groupedItems?: Item[]
  columnName: string
  authorName: string
  getColumnName?: (columnId: string) => string
  getAuthorName?: (authorId: string) => string
  onVote: () => void
  onUnvote: () => void
  onUpdate: (content: string) => void
  onDelete: () => void
  color: string
}

// Helper to get trigram from name
const getTrigram = (name: string): string => {
  if (!name) return '???'
  // Split by space and get initials, or first 3 chars if single word
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 3) {
    return parts.slice(0, 3).map(p => p[0]?.toUpperCase() || '').join('')
  } else if (parts.length === 2) {
    return (parts[0][0] + parts[1].slice(0, 2)).toUpperCase()
  }
  return name.slice(0, 3).toUpperCase()
}

// Generate obfuscated content with dashes
const obfuscateContent = (content: string): string => {
  // Replace each word with dashes of similar length
  return content.split(/\s+/).map(word => '—'.repeat(Math.max(2, word.length))).join(' ')
}

export default function ItemCard({
  item,
  canVote,
  canEdit,
  canDelete,
  canGroup,
  isObfuscated = false,
  groupedItems = [],
  columnName,
  authorName,
  getColumnName,
  getAuthorName,
  onVote,
  onUnvote,
  onUpdate,
  onDelete,
  color,
}: ItemCardProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState(item.content)
  const [hasVoted, setHasVoted] = useState(false)
  const [showGrouped, setShowGrouped] = useState(false)

  // Drag functionality
  const { attributes, listeners, setNodeRef: setDragRef, transform, isDragging } = useDraggable({
    id: item.id,
    disabled: !canGroup,
    data: { item },
  })

  // Drop functionality (item can be dropped onto)
  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `drop-${item.id}`,
    disabled: !canGroup,
    data: { item },
  })

  const style = transform ? {
    transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
  } : undefined

  const handleSave = () => {
    if (editContent.trim() && editContent !== item.content) {
      onUpdate(editContent.trim())
    }
    setIsEditing(false)
  }

  const handleCancel = () => {
    setEditContent(item.content)
    setIsEditing(false)
  }

  const handleVoteClick = () => {
    if (hasVoted) {
      onUnvote()
      setHasVoted(false)
    } else {
      onVote()
      setHasVoted(true)
    }
  }

  return (
    <div
      ref={setDropRef}
      className={clsx(
        'bg-white rounded-lg border border-gray-200 p-3 shadow-sm transition-all',
        isDragging && 'opacity-50 shadow-lg scale-105 z-50',
        isOver && 'ring-2 ring-primary-500 ring-offset-2 bg-primary-50',
        !isDragging && 'hover:shadow-md'
      )}
      style={{
        borderLeftWidth: '4px',
        borderLeftColor: color,
        ...style
      }}
    >
      {/* Drag handle - only visible in group phase */}
      {canGroup && (
        <div
          ref={setDragRef}
          {...listeners}
          {...attributes}
          className="flex items-center justify-center mb-2 -mt-1 cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600 border-b border-gray-100 pb-1"
        >
          <GripVertical className="w-4 h-4 rotate-90" />
          <span className="text-xs ml-1">Glisser pour grouper</span>
        </div>
      )}

      {isEditing ? (
        <div className="space-y-2">
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            className="w-full px-2 py-1 text-sm border border-gray-300 rounded resize-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            rows={3}
            autoFocus
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={handleCancel}
              className="p-1 text-gray-500 hover:text-gray-700 rounded"
            >
              <X className="w-4 h-4" />
            </button>
            <button
              onClick={handleSave}
              className="p-1 text-green-600 hover:text-green-700 rounded"
            >
              <Check className="w-4 h-4" />
            </button>
          </div>
        </div>
      ) : (
        <>
          {/* Labels */}
          <div className="flex flex-wrap gap-1 mb-2">
            <span
              className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium"
              style={{ backgroundColor: `${color}20`, color: color }}
            >
              {columnName}
            </span>
            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
              {getTrigram(authorName)}
            </span>
          </div>
          <p className={clsx(
            'text-sm whitespace-pre-wrap',
            isObfuscated ? 'text-gray-400 select-none' : 'text-gray-800'
          )}>
            {isObfuscated ? obfuscateContent(item.content) : item.content}
          </p>

          <div className="flex items-center justify-between mt-3 pt-2 border-t border-gray-100">
            {/* Vote button */}
            {canVote ? (
              <button
                onClick={handleVoteClick}
                className={clsx(
                  'flex items-center gap-1.5 px-2 py-1 rounded text-sm transition-colors',
                  hasVoted
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-500 hover:bg-gray-100'
                )}
              >
                <ThumbsUp className="w-4 h-4" />
                <span>{item.voteCount}</span>
              </button>
            ) : (
              <div className="flex items-center gap-1.5 text-gray-500 text-sm">
                <ThumbsUp className="w-4 h-4" />
                <span>{item.voteCount}</span>
              </div>
            )}

            {/* Edit/Delete buttons */}
            <div className="flex items-center gap-1">
              {canEdit && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="p-1 text-gray-400 hover:text-gray-600 rounded"
                >
                  <Edit2 className="w-4 h-4" />
                </button>
              )}
              {canDelete && (
                <button
                  onClick={() => {
                    if (confirm('Delete this item?')) {
                      onDelete()
                    }
                  }}
                  className="p-1 text-gray-400 hover:text-red-600 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>

          {/* Grouped items indicator */}
          {groupedItems.length > 0 && (
            <div className="mt-2 pt-2 border-t border-gray-100">
              <button
                onClick={() => setShowGrouped(!showGrouped)}
                className="flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700"
              >
                <Layers className="w-3 h-3" />
                <span>{groupedItems.length} item{groupedItems.length > 1 ? 's' : ''} groupé{groupedItems.length > 1 ? 's' : ''}</span>
              </button>
              {showGrouped && (
                <div className="mt-2 pl-3 border-l-2 border-gray-200 space-y-2">
                  {groupedItems.map((grouped) => {
                    const groupedColumnName = getColumnName?.(grouped.columnId) || grouped.columnId
                    const groupedAuthorName = getAuthorName?.(grouped.authorId) || 'Unknown'
                    return (
                      <div key={grouped.id} className="text-xs">
                        <div className="flex flex-wrap gap-1 mb-1">
                          <span className="inline-flex items-center px-1 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                            {groupedColumnName}
                          </span>
                          <span className="inline-flex items-center px-1 py-0.5 rounded text-xs font-medium bg-gray-50 text-gray-500">
                            {getTrigram(groupedAuthorName)}
                          </span>
                        </div>
                        <p className="text-gray-600">{grouped.content}</p>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}
