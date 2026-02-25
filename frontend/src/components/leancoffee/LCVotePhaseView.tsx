import { useMemo } from 'react'
import { useRetroStore } from '../../store/retroStore'
import { ThumbsUp } from 'lucide-react'
import clsx from 'clsx'

interface LCVotePhaseViewProps {
  send: (type: string, payload: Record<string, unknown>) => void
  isFacilitator: boolean
}

export default function LCVotePhaseView({ send, isFacilitator }: LCVotePhaseViewProps) {
  const { items, retro, myVotesOnItems } = useRetroStore()

  const topics = useMemo(() => {
    const topLevel = items.filter(item => !item.groupId)
    return topLevel.sort((a, b) => b.voteCount - a.voteCount)
  }, [items])

  const maxVotesPerUser = retro?.maxVotesPerUser || 5
  const maxVotesPerItem = retro?.maxVotesPerItem || 3

  const totalMyVotes = useMemo(() => {
    let total = 0
    myVotesOnItems.forEach(count => { total += count })
    return total
  }, [myVotesOnItems])

  const handleVote = (itemId: string) => {
    if (totalMyVotes >= maxVotesPerUser) return
    const votesOnItem = myVotesOnItems.get(itemId) || 0
    if (votesOnItem >= maxVotesPerItem) return
    send('vote_add', { itemId })
  }

  const handleUnvote = (itemId: string) => {
    const votesOnItem = myVotesOnItems.get(itemId) || 0
    if (votesOnItem <= 0) return
    send('vote_remove', { itemId })
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="text-center mb-6">
        <ThumbsUp className="w-8 h-8 text-primary-600 mx-auto mb-2" />
        <h2 className="text-xl font-semibold text-gray-900">Votez pour les sujets</h2>
        <p className="text-sm text-gray-500 mt-1">
          Votes restants : {maxVotesPerUser - totalMyVotes} / {maxVotesPerUser}
        </p>
      </div>

      <div className="space-y-3">
        {topics.map((topic) => {
          const myVotes = myVotesOnItems.get(topic.id) || 0
          const canVote = totalMyVotes < maxVotesPerUser && myVotes < maxVotesPerItem

          return (
            <div
              key={topic.id}
              className="flex items-center gap-3 p-4 bg-white rounded-lg border border-gray-200 shadow-sm"
            >
              <div className="flex-1">
                <p className="text-sm text-gray-800 whitespace-pre-wrap">{topic.content}</p>
              </div>

              <div className="flex items-center gap-2">
                {/* Vote count badge */}
                <span className={clsx(
                  'inline-flex items-center gap-1 px-2 py-1 rounded-full text-sm font-medium',
                  topic.voteCount > 0 ? 'bg-primary-100 text-primary-700' : 'bg-gray-100 text-gray-500'
                )}>
                  <ThumbsUp className="w-3.5 h-3.5" />
                  {topic.voteCount}
                </span>

                {/* Vote buttons */}
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleUnvote(topic.id)}
                    disabled={myVotes <= 0}
                    className="p-1.5 text-gray-400 hover:text-red-500 rounded disabled:opacity-30 disabled:cursor-not-allowed"
                    title="Retirer un vote"
                  >
                    <span className="text-sm font-bold">−</span>
                  </button>
                  <span className="text-xs text-gray-500 w-4 text-center">{myVotes}</span>
                  <button
                    onClick={() => handleVote(topic.id)}
                    disabled={!canVote}
                    className="p-1.5 text-primary-600 hover:text-primary-700 rounded disabled:opacity-30 disabled:cursor-not-allowed"
                    title="Ajouter un vote"
                  >
                    <span className="text-sm font-bold">+</span>
                  </button>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {isFacilitator && (
        <div className="mt-6 text-center">
          <button
            onClick={() => send('phase_next', {})}
            className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 text-sm font-medium"
          >
            Passer à la discussion
          </button>
        </div>
      )}
    </div>
  )
}
