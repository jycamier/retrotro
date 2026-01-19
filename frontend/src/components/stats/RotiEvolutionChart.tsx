import clsx from 'clsx'
import type { RotiEvolutionPoint } from '../../types'

interface RotiEvolutionChartProps {
  evolution: RotiEvolutionPoint[]
  className?: string
  showAverage?: boolean
  averageLabel?: string
}

export default function RotiEvolutionChart({
  evolution,
  className,
  showAverage = true,
  averageLabel = 'Moyenne équipe',
}: RotiEvolutionChartProps) {
  if (evolution.length === 0) {
    return (
      <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
        <h3 className="text-sm font-medium text-gray-700 mb-4">Évolution du ROTI</h3>
        <p className="text-center text-gray-500 text-sm py-8">Aucune donnée disponible</p>
      </div>
    )
  }

  const overallAverage = evolution.reduce((sum, p) => sum + p.average, 0) / evolution.length

  const getBarColor = (avg: number) => {
    if (avg >= 4) return 'bg-green-500'
    if (avg >= 3) return 'bg-lime-500'
    if (avg >= 2) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  return (
    <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-gray-700">Évolution du ROTI</h3>
        {showAverage && (
          <span className="text-xs text-gray-500">
            {averageLabel}: {overallAverage.toFixed(2)}
          </span>
        )}
      </div>

      <div className="relative">
        {/* Y-axis labels */}
        <div className="absolute left-0 top-0 bottom-6 w-8 flex flex-col justify-between text-xs text-gray-400">
          <span>5</span>
          <span>3</span>
          <span>1</span>
        </div>

        {/* Chart area */}
        <div className="ml-10 relative">
          {/* Grid lines */}
          <div className="absolute inset-0 flex flex-col justify-between pointer-events-none">
            {[5, 4, 3, 2, 1].map((level) => (
              <div key={level} className="border-t border-gray-100 w-full h-0" />
            ))}
          </div>

          {/* Average line */}
          {showAverage && (
            <div
              className="absolute left-0 right-0 border-t-2 border-dashed border-primary-400 pointer-events-none"
              style={{ top: `${((5 - overallAverage) / 4) * 100}%` }}
            />
          )}

          {/* Bars */}
          <div className="flex items-end gap-1 h-40">
            {evolution.map((point) => {
              const height = ((point.average - 1) / 4) * 100

              return (
                <div
                  key={point.retroId}
                  className="flex-1 flex flex-col items-center group relative"
                >
                  <div className="flex-1 w-full flex items-end justify-center">
                    <div
                      className={clsx(
                        'w-full max-w-12 rounded-t transition-all duration-300',
                        getBarColor(point.average),
                        'group-hover:opacity-80'
                      )}
                      style={{ height: `${height}%` }}
                    />
                  </div>

                  {/* Tooltip */}
                  <div className="absolute bottom-full mb-2 hidden group-hover:block z-10">
                    <div className="bg-gray-900 text-white text-xs rounded px-2 py-1 whitespace-nowrap">
                      <div className="font-medium">{point.retroName}</div>
                      <div>ROTI: {point.average.toFixed(2)}</div>
                      <div>{point.voteCount} vote{point.voteCount > 1 ? 's' : ''}</div>
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
                className="flex-1 text-center text-xs text-gray-400 truncate"
                title={point.retroName}
              >
                {index + 1}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-4 text-xs text-gray-500 text-center">
        Rétros du plus ancien au plus récent
      </div>
    </div>
  )
}
