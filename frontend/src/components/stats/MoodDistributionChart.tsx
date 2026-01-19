import clsx from 'clsx'
import type { MoodWeather } from '../../types'

interface MoodDistributionChartProps {
  distribution: Partial<Record<MoodWeather, number>>
  className?: string
}

const MOOD_CONFIG: Record<MoodWeather, { label: string; emoji: string; color: string }> = {
  sunny: { label: 'Ensoleillé', emoji: '☀️', color: 'bg-yellow-400' },
  partly_cloudy: { label: 'Partiellement nuageux', emoji: '⛅', color: 'bg-blue-300' },
  cloudy: { label: 'Nuageux', emoji: '☁️', color: 'bg-gray-400' },
  rainy: { label: 'Pluvieux', emoji: '🌧️', color: 'bg-blue-500' },
  stormy: { label: 'Orageux', emoji: '⛈️', color: 'bg-purple-600' },
}

const MOOD_ORDER: MoodWeather[] = ['sunny', 'partly_cloudy', 'cloudy', 'rainy', 'stormy']

export default function MoodDistributionChart({ distribution, className }: MoodDistributionChartProps) {
  const total = Object.values(distribution).reduce((sum, count) => sum + count, 0)
  const maxCount = Math.max(...Object.values(distribution), 1)

  return (
    <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
      <h3 className="text-sm font-medium text-gray-700 mb-4">Distribution des humeurs</h3>
      <div className="space-y-3">
        {MOOD_ORDER.map((mood) => {
          const config = MOOD_CONFIG[mood]
          const count = distribution[mood] || 0
          const percentage = total > 0 ? (count / total) * 100 : 0
          const barWidth = (count / maxCount) * 100

          return (
            <div key={mood} className="flex items-center gap-3">
              <div className="w-8 text-center text-lg">{config.emoji}</div>
              <div className="w-20 text-xs text-gray-600 truncate" title={config.label}>
                {config.label}
              </div>
              <div className="flex-1 h-6 bg-gray-100 rounded overflow-hidden">
                <div
                  className={clsx('h-full transition-all duration-500', config.color)}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-16 text-right text-xs text-gray-600">
                {count} ({percentage.toFixed(0)}%)
              </div>
            </div>
          )
        })}
      </div>
      {total === 0 && (
        <p className="text-center text-gray-500 text-sm mt-4">Aucune donnée disponible</p>
      )}
    </div>
  )
}
