import { useState, useMemo } from 'react'
import { X, ChevronLeft, ChevronRight, ThumbsUp, MessageSquare, Layers } from 'lucide-react'
import type { Item, Template } from '../../types'
import clsx from 'clsx'

interface DiscussionCarouselProps {
  items: Item[]
  template: Template
  isOpen: boolean
  onClose: () => void
  getAuthorName: (authorId: string) => string
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
  isOpen,
  onClose,
  getAuthorName,
}: DiscussionCarouselProps) {
  const [currentIndex, setCurrentIndex] = useState(0)

  // Get top-level items (not grouped under another item), sorted by votes
  const discussionItems = useMemo(() => {
    const topLevelItems = items.filter(item => !item.groupId)
    return topLevelItems.sort((a, b) => {
      // Calculate total votes including grouped items
      const aGroupedItems = items.filter(i => i.groupId === a.id)
      const bGroupedItems = items.filter(i => i.groupId === b.id)
      const aTotalVotes = a.voteCount + aGroupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      const bTotalVotes = b.voteCount + bGroupedItems.reduce((sum, i) => sum + i.voteCount, 0)
      return bTotalVotes - aTotalVotes
    })
  }, [items])

  // Get grouped items for current discussion item
  const getGroupedItems = (parentId: string): Item[] => {
    return items.filter(item => item.groupId === parentId)
  }

  // Get column info
  const getColumnInfo = (columnId: string) => {
    return template.columns.find(c => c.id === columnId)
  }

  // Calculate total votes for an item including its grouped items
  const getTotalVotes = (item: Item): number => {
    const groupedItems = getGroupedItems(item.id)
    return item.voteCount + groupedItems.reduce((sum, i) => sum + i.voteCount, 0)
  }

  const currentItem = discussionItems[currentIndex]
  const groupedItems = currentItem ? getGroupedItems(currentItem.id) : []
  const totalItems = discussionItems.length

  const goToPrevious = () => {
    setCurrentIndex(prev => (prev > 0 ? prev - 1 : totalItems - 1))
  }

  const goToNext = () => {
    setCurrentIndex(prev => (prev < totalItems - 1 ? prev + 1 : 0))
  }

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowLeft') goToPrevious()
    if (e.key === 'ArrowRight') goToNext()
    if (e.key === 'Escape') onClose()
  }

  if (!isOpen || !currentItem) return null

  const column = getColumnInfo(currentItem.columnId)
  const totalVotes = getTotalVotes(currentItem)

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      onClick={onClose}
      onKeyDown={handleKeyDown}
      tabIndex={0}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 bg-gray-50">
          <div className="flex items-center gap-3">
            <MessageSquare className="w-5 h-5 text-primary-600" />
            <h2 className="text-lg font-semibold text-gray-900">Discussion</h2>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500">
              {currentIndex + 1} / {totalItems}
            </span>
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-6">
          {/* Vote count badge */}
          <div className="flex items-center justify-center mb-6">
            <div className="flex items-center gap-2 px-4 py-2 bg-primary-50 text-primary-700 rounded-full">
              <ThumbsUp className="w-5 h-5" />
              <span className="text-lg font-semibold">{totalVotes} vote{totalVotes !== 1 ? 's' : ''}</span>
            </div>
          </div>

          {/* Main item card */}
          <div
            className="bg-white border-2 rounded-xl p-6 mb-4"
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
            <div className="mt-6">
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

        {/* Navigation */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-gray-200 bg-gray-50">
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
            {discussionItems.slice(0, 10).map((_, index) => (
              <button
                key={index}
                onClick={() => setCurrentIndex(index)}
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
    </div>
  )
}
