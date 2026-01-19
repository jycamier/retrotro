import { useEffect, useState } from 'react'
import { useRetroStore } from '../../store/retroStore'
import { Play, Pause, Plus } from 'lucide-react'
import clsx from 'clsx'

interface PhaseTimerProps {
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

export default function PhaseTimer({ isFacilitator, send }: PhaseTimerProps) {
  const { timerEndAt, isTimerRunning, timerRemainingSeconds } = useRetroStore()
  const [displayTime, setDisplayTime] = useState(0)

  useEffect(() => {
    if (!isTimerRunning || !timerEndAt) {
      setDisplayTime(timerRemainingSeconds)
      return
    }

    const updateTime = () => {
      const now = new Date()
      const diff = Math.max(0, Math.floor((timerEndAt.getTime() - now.getTime()) / 1000))
      setDisplayTime(diff)
    }

    updateTime()
    const interval = setInterval(updateTime, 1000)
    return () => clearInterval(interval)
  }, [timerEndAt, isTimerRunning, timerRemainingSeconds])

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const isWarning = displayTime <= 60 && displayTime > 30
  const isCritical = displayTime <= 30 && displayTime > 0

  const handleStart = () => {
    send('timer_start', { duration_seconds: 300 }) // Default 5 minutes
  }

  const handlePause = () => {
    send('timer_pause', {})
  }

  const handleResume = () => {
    send('timer_resume', {})
  }

  const handleAddTime = () => {
    send('timer_add_time', { seconds: 60 })
  }

  return (
    <div className="flex items-center gap-3">
      {/* Timer Display */}
      <div
        className={clsx(
          'px-4 py-2 rounded-lg font-mono text-lg font-semibold',
          isTimerRunning && isCritical && 'bg-red-100 text-red-700 animate-pulse',
          isTimerRunning && isWarning && 'bg-yellow-100 text-yellow-700',
          isTimerRunning && !isWarning && !isCritical && 'bg-green-100 text-green-700',
          !isTimerRunning && 'bg-gray-100 text-gray-700'
        )}
      >
        {formatTime(displayTime)}
      </div>

      {/* Facilitator Controls */}
      {isFacilitator && (
        <div className="flex items-center gap-1">
          {!isTimerRunning ? (
            <button
              onClick={timerRemainingSeconds > 0 ? handleResume : handleStart}
              className="p-2 text-green-600 hover:bg-green-50 rounded-lg"
              title="Start timer"
            >
              <Play className="w-5 h-5" />
            </button>
          ) : (
            <button
              onClick={handlePause}
              className="p-2 text-yellow-600 hover:bg-yellow-50 rounded-lg"
              title="Pause timer"
            >
              <Pause className="w-5 h-5" />
            </button>
          )}

          <button
            onClick={handleAddTime}
            className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg"
            title="Add 1 minute"
          >
            <Plus className="w-5 h-5" />
          </button>
        </div>
      )}
    </div>
  )
}
