import clsx from 'clsx'

interface RotiDistributionChartProps {
  distribution: Record<number, number>
  className?: string
}

const ROTI_LABELS = ['1 - Perte de temps', '2 - Peu utile', '3 - Correct', '4 - Utile', '5 - Excellent']
const ROTI_COLORS = ['bg-red-500', 'bg-orange-500', 'bg-yellow-500', 'bg-lime-500', 'bg-green-500']

export default function RotiDistributionChart({ distribution, className }: RotiDistributionChartProps) {
  const total = Object.values(distribution).reduce((sum, count) => sum + count, 0)
  const maxCount = Math.max(...Object.values(distribution), 1)

  return (
    <div className={clsx('bg-white rounded-lg border border-gray-200 p-4', className)}>
      <h3 className="text-sm font-medium text-gray-700 mb-4">Distribution des votes ROTI</h3>
      <div className="space-y-3">
        {[1, 2, 3, 4, 5].map((rating, index) => {
          const count = distribution[rating] || 0
          const percentage = total > 0 ? (count / total) * 100 : 0
          const barWidth = (count / maxCount) * 100

          return (
            <div key={rating} className="flex items-center gap-3">
              <div className="w-24 text-xs text-gray-600 truncate" title={ROTI_LABELS[index]}>
                {rating} - {ROTI_LABELS[index].split(' - ')[1]}
              </div>
              <div className="flex-1 h-6 bg-gray-100 rounded overflow-hidden">
                <div
                  className={clsx('h-full transition-all duration-500', ROTI_COLORS[index])}
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
