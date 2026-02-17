import { Loader2, Wifi, WifiOff, CheckCircle } from 'lucide-react'

interface LoadingScreenProps {
  isConnected: boolean
  isStateLoaded: boolean
  retro: unknown
  template: unknown
  connectionError: string | null
  onRetry: () => void
}

export default function LoadingScreen({
  isConnected,
  isStateLoaded,
  retro,
  template,
  connectionError,
  onRetry,
}: LoadingScreenProps) {
  if (connectionError) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
        <div className="text-center bg-white rounded-2xl shadow-xl p-8 max-w-md mx-4">
          <div className="text-red-500 mb-4">
            <WifiOff className="w-16 h-16 mx-auto" />
          </div>
          <h2 className="text-xl font-semibold text-gray-900 mb-2">
            Connexion perdue
          </h2>
          <p className="text-red-600 mb-6">{connectionError}</p>
          <button
            onClick={onRetry}
            className="px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 font-medium transition-colors"
          >
            Rafraîchir la page
          </button>
        </div>
      </div>
    )
  }

  const steps = [
    {
      id: 'connect',
      label: 'Connexion au serveur',
      isActive: !isConnected,
      isComplete: isConnected,
    },
    {
      id: 'load',
      label: 'Chargement des données',
      isActive: isConnected && !isStateLoaded,
      isComplete: isStateLoaded,
    },
    {
      id: 'prepare',
      label: 'Préparation de la rétrospective',
      isActive: isStateLoaded && (!retro || !template),
      isComplete: !!(retro && template),
    },
  ]

  const progress = Math.max(0, Math.min(100, (steps.filter(s => s.isComplete).length / steps.length) * 100))

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
      <div className="text-center bg-white rounded-2xl shadow-xl p-8 max-w-md mx-4 w-full">
        {/* Logo */}
        <div className="mb-8">
          <img
            src="/logo.png"
            alt="Retrotro"
            className="w-24 h-24 mx-auto animate-pulse"
          />
        </div>

        {/* Titre */}
        <h1 className="text-2xl font-bold text-gray-900 mb-2">
          Retrotro
        </h1>
        <p className="text-gray-600 mb-8">
          Connexion en cours...
        </p>

        {/* Barre de progression */}
        <div className="mb-8">
          <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-primary-600 transition-all duration-500 ease-out"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {/* Étapes */}
        <div className="space-y-4">
          {steps.map((step) => (
            <div
              key={step.id}
              className={`flex items-center gap-3 transition-all duration-300 ${
                step.isComplete
                  ? 'opacity-100'
                  : step.isActive
                  ? 'opacity-100'
                  : 'opacity-40'
              }`}
            >
              <div className="flex-shrink-0 w-6">
                {step.isComplete ? (
                  <CheckCircle className="w-5 h-5 text-green-500" />
                ) : step.isActive ? (
                  <Loader2 className="w-5 h-5 text-primary-600 animate-spin" />
                ) : (
                  <div className="w-5 h-5 rounded-full border-2 border-gray-300" />
                )}
              </div>
              <span
                className={`text-sm ${
                  step.isComplete
                    ? 'text-gray-400 line-through'
                    : step.isActive
                    ? 'text-gray-700 font-medium'
                    : 'text-gray-400'
                }`}
              >
                {step.label}
              </span>
            </div>
          ))}
        </div>

        {/* Info connexion */}
        <div className="mt-8 pt-6 border-t border-gray-200">
          <div className="flex items-center justify-center gap-2 text-sm">
            {isConnected ? (
              <>
                <Wifi className="w-4 h-4 text-green-500" />
                <span className="text-green-600">Connecté</span>
              </>
            ) : (
              <>
                <WifiOff className="w-4 h-4 text-gray-400" />
                <span className="text-gray-500">Connexion en cours...</span>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="mt-8 text-center text-sm text-gray-500">
        <p>"So so funny!" - Retrotro the Gopher</p>
      </div>
    </div>
  )
}
