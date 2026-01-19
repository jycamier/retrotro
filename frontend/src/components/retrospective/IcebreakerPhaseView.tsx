import { Sun, CloudSun, Cloud, CloudRain, CloudLightning, ArrowRight } from 'lucide-react'
import type { Participant, MoodWeather } from '../../types'
import clsx from 'clsx'

interface IcebreakerPhaseViewProps {
  moods: Map<string, MoodWeather>
  participants: Participant[]
  currentUserId: string
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

const moodConfig: Record<MoodWeather, { icon: typeof Sun; label: string; color: string; bgColor: string }> = {
  sunny: { icon: Sun, label: 'Ensoleillé', color: 'text-yellow-500', bgColor: 'bg-yellow-50 border-yellow-200' },
  partly_cloudy: { icon: CloudSun, label: 'Partiellement nuageux', color: 'text-orange-400', bgColor: 'bg-orange-50 border-orange-200' },
  cloudy: { icon: Cloud, label: 'Nuageux', color: 'text-gray-400', bgColor: 'bg-gray-50 border-gray-200' },
  rainy: { icon: CloudRain, label: 'Pluvieux', color: 'text-blue-400', bgColor: 'bg-blue-50 border-blue-200' },
  stormy: { icon: CloudLightning, label: 'Orageux', color: 'text-purple-500', bgColor: 'bg-purple-50 border-purple-200' },
}

const moodOrder: MoodWeather[] = ['sunny', 'partly_cloudy', 'cloudy', 'rainy', 'stormy']

export default function IcebreakerPhaseView({
  moods,
  participants,
  currentUserId,
  isFacilitator,
  send,
}: IcebreakerPhaseViewProps) {
  const currentUserMood = moods.get(currentUserId)
  const moodCount = moods.size
  const participantCount = participants.length

  const handleSelectMood = (mood: MoodWeather) => {
    send('mood_set', { mood })
  }

  const handleNextPhase = () => {
    send('phase_next', {})
  }

  // Get initials from name
  const getInitials = (name: string): string => {
    const parts = name.trim().split(/\s+/)
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase()
    }
    return name.slice(0, 2).toUpperCase()
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] p-8">
      <div className="max-w-2xl w-full">
        {/* Header */}
        <div className="text-center mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            Comment vous sentez-vous aujourd'hui ?
          </h2>
          <p className="text-gray-600">
            Partagez votre humeur avec l'équipe avant de commencer la rétrospective
          </p>
        </div>

        {/* Mood Selection */}
        <div className="flex justify-center gap-4 mb-8">
          {moodOrder.map((mood) => {
            const config = moodConfig[mood]
            const Icon = config.icon
            const isSelected = currentUserMood === mood

            return (
              <button
                key={mood}
                onClick={() => handleSelectMood(mood)}
                className={clsx(
                  'flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all',
                  isSelected
                    ? `${config.bgColor} ring-2 ring-offset-2 ring-primary-500`
                    : 'bg-white border-gray-200 hover:border-gray-300 hover:shadow-md'
                )}
              >
                <Icon className={clsx('w-10 h-10', config.color)} />
                <span className={clsx(
                  'text-sm font-medium',
                  isSelected ? 'text-gray-900' : 'text-gray-600'
                )}>
                  {config.label}
                </span>
              </button>
            )
          })}
        </div>

        {/* Progress indicator */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 rounded-full">
            <span className="text-sm text-gray-600">
              {moodCount}/{participantCount} participants ont partagé leur humeur
            </span>
          </div>
        </div>

        {/* Team moods grid */}
        {moodCount > 0 && (
          <div className="bg-white rounded-xl border border-gray-200 p-6 mb-8">
            <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">
              Humeur de l'équipe
            </h3>
            <div className="flex flex-wrap gap-3">
              {participants.map((participant) => {
                const mood = moods.get(participant.userId)
                if (!mood) return null

                const config = moodConfig[mood]
                const Icon = config.icon
                const isCurrentUser = participant.userId === currentUserId

                return (
                  <div
                    key={participant.userId}
                    className={clsx(
                      'flex items-center gap-2 px-3 py-2 rounded-lg border',
                      config.bgColor,
                      isCurrentUser && 'ring-2 ring-primary-400'
                    )}
                    title={participant.name}
                  >
                    <span className="text-sm font-medium text-gray-700">
                      {getInitials(participant.name)}
                    </span>
                    <Icon className={clsx('w-5 h-5', config.color)} />
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Waiting for others */}
        {moodCount > 0 && moodCount < participantCount && (
          <div className="text-center mb-8">
            <p className="text-sm text-gray-500">
              En attente de {participantCount - moodCount} participant{participantCount - moodCount > 1 ? 's' : ''}...
            </p>
          </div>
        )}

        {/* Facilitator controls */}
        {isFacilitator && (
          <div className="flex justify-center">
            <button
              onClick={handleNextPhase}
              className="flex items-center gap-2 px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium"
            >
              Continuer vers Brainstorm
              <ArrowRight className="w-5 h-5" />
            </button>
          </div>
        )}

        {/* Non-facilitator waiting message */}
        {!isFacilitator && currentUserMood && (
          <div className="text-center">
            <p className="text-sm text-gray-500">
              En attente que le facilitateur passe à la phase suivante...
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
