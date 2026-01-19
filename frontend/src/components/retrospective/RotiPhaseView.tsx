import { useState } from 'react'
import { Star, Eye, Check, Flag } from 'lucide-react'
import type { Participant, RotiResults } from '../../types'
import clsx from 'clsx'

interface RotiPhaseViewProps {
  rotiVotedUserIds: Set<string>
  rotiResults: RotiResults | null
  participants: Participant[]
  currentUserId: string
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

const ratingLabels: Record<number, string> = {
  1: 'Perte de temps',
  2: 'Peu utile',
  3: 'Correct',
  4: 'Utile',
  5: 'Excellent',
}

export default function RotiPhaseView({
  rotiVotedUserIds,
  rotiResults,
  participants,
  currentUserId,
  isFacilitator,
  send,
}: RotiPhaseViewProps) {
  const [selectedRating, setSelectedRating] = useState<number | null>(null)
  const [showEndConfirm, setShowEndConfirm] = useState(false)

  const hasVoted = rotiVotedUserIds.has(currentUserId)
  const voteCount = rotiVotedUserIds.size
  const participantCount = participants.length
  const isRevealed = rotiResults?.revealed || false

  const handleVote = (rating: number) => {
    setSelectedRating(rating)
    send('roti_vote', { rating })
  }

  const handleRevealResults = () => {
    send('roti_reveal', {})
  }

  const handleEndRetro = () => {
    send('retro_end', {})
    setShowEndConfirm(false)
  }

  // Calculate max count for distribution bar widths
  const maxCount = rotiResults?.distribution
    ? Math.max(...Object.values(rotiResults.distribution), 1)
    : 1

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] p-8">
      <div className="max-w-xl w-full">
        {/* Header */}
        <div className="text-center mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            ROTI - Return On Time Invested
          </h2>
          <p className="text-gray-600">
            {isRevealed
              ? 'Voici les résultats de cette rétrospective'
              : 'Notez cette rétrospective de 1 à 5'}
          </p>
        </div>

        {/* Voting phase */}
        {!isRevealed && (
          <>
            {/* Rating selection */}
            {!hasVoted ? (
              <div className="bg-white rounded-xl border border-gray-200 p-6 mb-6">
                <div className="flex justify-center gap-2 mb-6">
                  {[1, 2, 3, 4, 5].map((rating) => (
                    <button
                      key={rating}
                      onClick={() => handleVote(rating)}
                      className={clsx(
                        'flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all min-w-[80px]',
                        selectedRating === rating
                          ? 'border-yellow-400 bg-yellow-50 ring-2 ring-yellow-200'
                          : 'border-gray-200 hover:border-yellow-300 hover:bg-yellow-50'
                      )}
                    >
                      <div className="flex gap-0.5">
                        {Array.from({ length: rating }).map((_, i) => (
                          <Star
                            key={i}
                            className={clsx(
                              'w-5 h-5',
                              selectedRating === rating
                                ? 'text-yellow-500 fill-yellow-500'
                                : 'text-yellow-400 fill-yellow-400'
                            )}
                          />
                        ))}
                      </div>
                      <span className="text-xs font-medium text-gray-600">
                        {rating}
                      </span>
                    </button>
                  ))}
                </div>
                <div className="text-center">
                  <p className="text-sm text-gray-500">
                    Cliquez sur une note pour voter
                  </p>
                </div>
              </div>
            ) : (
              <div className="bg-green-50 rounded-xl border border-green-200 p-6 mb-6 text-center">
                <div className="flex items-center justify-center gap-2 text-green-700 mb-2">
                  <Check className="w-5 h-5" />
                  <span className="font-medium">Vote enregistré !</span>
                </div>
                <p className="text-sm text-green-600">
                  Merci pour votre participation
                </p>
              </div>
            )}

            {/* Rating labels */}
            <div className="grid grid-cols-5 gap-2 mb-6 text-center text-xs text-gray-500">
              {[1, 2, 3, 4, 5].map((rating) => (
                <div key={rating}>{ratingLabels[rating]}</div>
              ))}
            </div>

            {/* Progress */}
            <div className="text-center mb-6">
              <div className="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 rounded-full">
                <span className="text-sm text-gray-600">
                  {voteCount}/{participantCount} votes
                </span>
              </div>
            </div>

            {/* Facilitator reveal button */}
            {isFacilitator && voteCount > 0 && (
              <div className="flex justify-center">
                <button
                  onClick={handleRevealResults}
                  className="flex items-center gap-2 px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium"
                >
                  <Eye className="w-5 h-5" />
                  Révéler les résultats
                </button>
              </div>
            )}

            {/* Non-facilitator waiting message */}
            {!isFacilitator && hasVoted && (
              <div className="text-center">
                <p className="text-sm text-gray-500">
                  En attente de la révélation des résultats par le facilitateur...
                </p>
              </div>
            )}
          </>
        )}

        {/* Results phase */}
        {isRevealed && rotiResults && (
          <>
            {/* Average score */}
            <div className="bg-white rounded-xl border border-gray-200 p-8 mb-6 text-center">
              <div className="text-6xl font-bold text-gray-900 mb-2">
                {rotiResults.average.toFixed(1)}
              </div>
              <div className="flex justify-center gap-1 mb-4">
                {[1, 2, 3, 4, 5].map((star) => (
                  <Star
                    key={star}
                    className={clsx(
                      'w-8 h-8',
                      star <= Math.round(rotiResults.average)
                        ? 'text-yellow-500 fill-yellow-500'
                        : 'text-gray-300'
                    )}
                  />
                ))}
              </div>
              <p className="text-gray-500">
                sur 5 ({rotiResults.totalVotes} vote{rotiResults.totalVotes > 1 ? 's' : ''})
              </p>
            </div>

            {/* Distribution chart */}
            <div className="bg-white rounded-xl border border-gray-200 p-6 mb-6">
              <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">
                Distribution des votes
              </h3>
              <div className="space-y-3">
                {[5, 4, 3, 2, 1].map((rating) => {
                  const count = rotiResults.distribution[rating] || 0
                  const percentage = count / maxCount * 100

                  return (
                    <div key={rating} className="flex items-center gap-3">
                      <div className="flex items-center gap-1 w-16">
                        <span className="text-sm font-medium text-gray-700">{rating}</span>
                        <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />
                      </div>
                      <div className="flex-1 bg-gray-100 rounded-full h-6 overflow-hidden">
                        <div
                          className="bg-yellow-400 h-full rounded-full transition-all duration-500"
                          style={{ width: `${percentage}%` }}
                        />
                      </div>
                      <span className="text-sm text-gray-500 w-8 text-right">
                        {count}
                      </span>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Facilitator end retro button */}
            {isFacilitator && (
              <div className="flex justify-center">
                <button
                  onClick={() => setShowEndConfirm(true)}
                  className="flex items-center gap-2 px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors font-medium"
                >
                  <Flag className="w-5 h-5" />
                  Terminer la rétrospective
                </button>
              </div>
            )}

            {/* Non-facilitator message */}
            {!isFacilitator && (
              <div className="text-center">
                <p className="text-sm text-gray-500">
                  En attente que le facilitateur termine la rétrospective...
                </p>
              </div>
            )}
          </>
        )}
      </div>

      {/* End Confirmation Dialog */}
      {showEndConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-full bg-green-100 flex items-center justify-center">
                <Flag className="w-5 h-5 text-green-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900">
                Terminer la rétrospective ?
              </h3>
            </div>
            <p className="text-gray-600 mb-6">
              Cette action mettra fin à la rétrospective. Un résumé sera affiché avec toutes les actions créées.
            </p>
            {rotiResults && (
              <div className="flex items-center gap-3 mb-4 p-3 bg-gray-50 rounded-lg">
                <span className="text-sm text-gray-500">Score ROTI :</span>
                <span className="font-bold text-gray-900">{rotiResults.average.toFixed(1)}/5</span>
              </div>
            )}
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowEndConfirm(false)}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200"
              >
                Annuler
              </button>
              <button
                onClick={handleEndRetro}
                className="px-4 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700"
              >
                Confirmer
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
