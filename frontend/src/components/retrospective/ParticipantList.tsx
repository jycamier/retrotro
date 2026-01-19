import type { Participant } from '../../types'
import { Crown } from 'lucide-react'
import clsx from 'clsx'

interface ParticipantListProps {
  participants: Participant[]
  facilitatorId: string
  compact?: boolean
}

// Get initials/trigram from name
const getInitials = (name: string): string => {
  if (!name) return '??'
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.slice(0, 2).toUpperCase()
}

export default function ParticipantList({
  participants,
  facilitatorId,
  compact = false,
}: ParticipantListProps) {
  if (compact) {
    return (
      <div>
        <h3 className="text-xs font-medium text-gray-500 uppercase mb-2">
          Participants ({participants.length})
        </h3>
        <div className="flex flex-wrap gap-1.5">
          {participants.map((participant) => {
            const isFacilitator = participant.userId === facilitatorId
            return (
              <div
                key={participant.userId}
                className={clsx(
                  'flex items-center gap-1 px-2 py-1 rounded text-xs font-medium',
                  isFacilitator ? 'bg-yellow-100 text-yellow-800' : 'bg-gray-100 text-gray-700'
                )}
                title={participant.name}
              >
                {getInitials(participant.name)}
                {isFacilitator && <Crown className="w-3 h-3 text-yellow-600" />}
              </div>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-3">Participants</h3>
      <div className="space-y-2">
        {participants.map((participant) => (
          <div
            key={participant.userId}
            className="flex items-center gap-2 p-2 bg-gray-50 rounded-lg"
          >
            <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-xs font-medium text-primary-700">
              {getInitials(participant.name)}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900 truncate">
                {participant.name}
              </p>
            </div>
            {participant.userId === facilitatorId && (
              <span title="Facilitateur">
                <Crown className="w-4 h-4 text-yellow-500" />
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
