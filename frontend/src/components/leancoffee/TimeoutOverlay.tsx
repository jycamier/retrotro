import { useEffect, useMemo } from 'react'
import './TimeoutOverlay.css'

interface TimeoutOverlayProps {
  onDismiss: () => void
}

export default function TimeoutOverlay({ onDismiss }: TimeoutOverlayProps) {
  const particles = useMemo(
    () =>
      Array.from({ length: 20 }, (_, i) => ({
        id: i,
        left: Math.random() * 100,
        delay: Math.random() * 10,
        duration: 8 + Math.random() * 6,
      })),
    []
  )

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' || e.key === 'Enter' || e.key === ' ') {
        onDismiss()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onDismiss])

  return (
    <div className="timeout-overlay" onClick={onDismiss}>
      {/* Grid background */}
      <div className="grid-bg" />

      {/* Scanlines */}
      <div className="scanlines" />

      {/* CRT vignette */}
      <div className="crt-screen" />

      {/* Particles */}
      <div className="particles">
        {particles.map((p) => (
          <div
            key={p.id}
            className="particle"
            style={{
              left: `${p.left}%`,
              animationDelay: `${p.delay}s`,
              animationDuration: `${p.duration}s`,
            }}
          />
        ))}
      </div>

      {/* Nedry */}
      <div className="nedry-container">
        <div className="nedry-head">
          <div className="nedry-hair" />
          <div className="glasses">
            <div className="glass-lens" />
            <div className="glasses-bridge" />
            <div className="glass-lens" />
          </div>
          <div className="nedry-mouth" />
        </div>
        <div className="nedry-body" />
      </div>

      {/* TIMEOUT text */}
      <div className="timeout-text-container">
        <h1 className="timeout-text">TIMEOUT</h1>
      </div>

      {/* Subtitle */}
      <p className="timeout-subtitle">[ TEMPS ECOULE - PLUS DE TEMPS ? ]</p>

      {/* Dismiss */}
      <button className="timeout-dismiss" onClick={onDismiss}>
        Continuer
      </button>
    </div>
  )
}
