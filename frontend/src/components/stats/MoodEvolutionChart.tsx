import clsx from 'clsx'
import type { MoodEvolutionPoint, MoodWeather } from '../../types'

interface MoodEvolutionChartProps {
  evolution: MoodEvolutionPoint[]
  className?: string
}

const MOOD_CONFIG: Record<MoodWeather, { emoji: string; color: string; order: number }> = {
  sunny: { emoji: '☀️', color: 'bg-yellow-400', order: 0 },
  partly_cloudy: { emoji: '⛅', color: 'bg-blue-300', order: 1 },
  cloudy: { emoji: '☁️', color: 'bg-gray-400', order: 2 },
  rainy: { emoji: '🌧️', color: 'bg-blue-500', order: 3 },
  stormy: { emoji: '⛈️', color: 'bg-purple-600', order: 4 },
}

const MOOD_ORDER: MoodWeather[] = ['sunny', 'partly_cloudy', 'cloudy', 'rainy', 'stormy']

export default function MoodEvolutionChart({ evolution, className }: MoodEvolutionChartProps) {
  if (evolution.length === 0) {
    return (
      <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
        <h3 className="text-sm font-medium text-gray-700 mb-4">Évolution des humeurs</h3>
        <p className="text-center text-gray-500 text-sm py-8">Aucune donnée disponible</p>
      </div>
    )
  }

  return (
    <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
      <h3 className="text-sm font-medium text-gray-700 mb-4">Évolution des humeurs</h3>

      {/* Legend */}
      <div className="flex flex-wrap gap-2 mb-4 justify-center">
        {MOOD_ORDER.map((mood) => (
          <div key={mood} className="flex items-center gap-1">
            <span className="text-sm">{MOOD_CONFIG[mood].emoji}</span>
          </div>
        ))}
      </div>

      {/* Stacked bar chart */}
      <div className="flex items-end gap-1 h-40">
        {evolution.map((point) => {
          const total = point.moodCount || 1

          return (
            <div
              key={point.retroId}
              className="flex-1 flex flex-col group relative"
            >
              <div className="flex-1 flex flex-col justify-end">
                <div className="w-full flex flex-col-reverse">
                  {MOOD_ORDER.map((mood) => {
                    const count = point.distribution?.[mood] || 0
                    const percentage = (count / total) * 100
                    if (percentage === 0) return null

                    return (
                      <div
                        key={mood}
                        className={clsx(MOOD_CONFIG[mood].color, 'w-full transition-all duration-300')}
                        style={{ height: `${(percentage / 100) * 160}px` }}
                      />
                    )
                  })}
                </div>
              </div>

              {/* Tooltip */}
              <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:block z-10">
                <div className="bg-gray-900 text-white text-xs rounded px-2 py-1 whitespace-nowrap">
                  <div className="font-medium">{point.retroName}</div>
                  <div className="mt-1">
                    {MOOD_ORDER.map((mood) => {
                      const count = point.distribution?.[mood] || 0
                      if (count === 0) return null
                      return (
                        <div key={mood}>
                          {MOOD_CONFIG[mood].emoji} {count}
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* X-axis labels */}
      <div className="flex gap-1 mt-2">
        {evolution.map((point, index) => (
          <div
            key={point.retroId}
            className="flex-1 text-center text-xs text-gray-400"
            title={point.retroName}
          >
            {index + 1}
          </div>
        ))}
      </div>

      <div className="mt-4 text-xs text-gray-500 text-center">
        Rétros du plus ancien au plus récent
      </div>
    </div>
  )
}
