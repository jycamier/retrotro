import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { authApi } from '../api/client'
import type { DevUser, DevUsersResponse } from '../types'
import { Shield, Users, User as UserIcon } from 'lucide-react'
import clsx from 'clsx'

// Role colors and badges
const roleConfig = {
  admin: {
    color: 'bg-violet-100 border-violet-300 hover:bg-violet-200',
    badge: 'bg-violet-500 text-white',
    icon: Shield,
    label: 'Admin',
  },
  member: {
    color: 'bg-gray-100 border-gray-300 hover:bg-gray-200',
    badge: 'bg-gray-500 text-white',
    icon: UserIcon,
    label: 'Membre',
  },
}

function DevUserCard({
  user,
  onLogin,
  loading
}: {
  user: DevUser
  onLogin: (user: DevUser) => void
  loading: boolean
}) {
  const config = roleConfig[user.teamRole]
  const Icon = config.icon

  return (
    <button
      onClick={() => onLogin(user)}
      disabled={loading}
      className={clsx(
        'w-full p-4 rounded-xl border-2 transition-all text-left',
        config.color,
        loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'
      )}
    >
      <div className="flex items-start gap-3">
        <div className={clsx(
          'w-12 h-12 rounded-full flex items-center justify-center text-lg font-bold',
          user.isAdmin ? 'bg-violet-500 text-white' : 'bg-primary-500 text-white'
        )}>
          {user.displayName.charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-gray-900 truncate">
              {user.displayName}
            </span>
            {user.isAdmin && (
              <span className="px-1.5 py-0.5 text-xs font-medium bg-violet-500 text-white rounded">
                ADMIN
              </span>
            )}
          </div>
          <p className="text-sm text-gray-600 truncate">{user.email}</p>
          <div className="mt-2 flex items-center gap-1.5">
            <span className={clsx('px-2 py-0.5 text-xs font-medium rounded flex items-center gap-1', config.badge)}>
              <Icon className="w-3 h-3" />
              {config.label}
            </span>
          </div>
        </div>
      </div>
    </button>
  )
}

export default function LoginPage() {
  const { isAuthenticated, setAuth } = useAuthStore()
  const navigate = useNavigate()
  const [loginInfo, setLoginInfo] = useState<{ oidcConfigured: boolean; devMode: boolean } | null>(null)
  const [devUsers, setDevUsers] = useState<DevUsersResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/')
    }
  }, [isAuthenticated, navigate])

  useEffect(() => {
    // Check login info on mount
    authApi.getLoginInfo()
      .then(info => {
        setLoginInfo(info)
        // If dev mode, fetch dev users
        if (info.devMode) {
          authApi.getDevUsers()
            .then(setDevUsers)
            .catch(err => console.error('Failed to fetch dev users:', err))
        }
      })
      .catch(() => setLoginInfo({ oidcConfigured: false, devMode: false }))
  }, [])

  const handleSSOLogin = () => {
    window.location.href = '/auth/login'
  }

  const handleDevLogin = async (user: DevUser) => {
    setLoading(true)
    setError(null)

    try {
      const result = await authApi.devLogin(user.email, user.displayName)
      setAuth(result.user, result.accessToken)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
      <div className="max-w-3xl w-full space-y-8 p-8 bg-white rounded-xl shadow-lg">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900">RetroTro</h1>
          <p className="mt-2 text-gray-600">
            Agile retrospective tool for effective team collaboration
          </p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        <div className="space-y-6">
          {loginInfo?.oidcConfigured && (
            <button
              onClick={handleSSOLogin}
              className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium"
            >
              Sign in with SSO
            </button>
          )}

          {loginInfo?.devMode && devUsers && (
            <>
              {loginInfo?.oidcConfigured && (
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-gray-300" />
                  </div>
                  <div className="relative flex justify-center text-sm">
                    <span className="px-2 bg-white text-gray-500">Or select a dev user</span>
                  </div>
                </div>
              )}

              <div className="space-y-4">
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Users className="w-4 h-4" />
                  <span>
                    Team: <strong>{devUsers.team.name}</strong>
                  </span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {devUsers.users.map((user) => (
                    <DevUserCard
                      key={user.id}
                      user={user}
                      onLogin={handleDevLogin}
                      loading={loading}
                    />
                  ))}
                </div>
              </div>

              <div className="text-center text-xs text-yellow-600 bg-yellow-50 p-2 rounded">
                Dev mode is enabled. In production, configure OIDC for secure authentication.
              </div>
            </>
          )}

          {loginInfo?.devMode && !devUsers && (
            <div className="text-center text-gray-500">
              Loading dev users...
            </div>
          )}

          {!loginInfo?.oidcConfigured && !loginInfo?.devMode && loginInfo !== null && (
            <div className="text-center text-red-600">
              Authentication not configured. Please set up OIDC or enable dev mode.
            </div>
          )}
        </div>

        <div className="text-center text-sm text-gray-500">
          <p>Secure authentication via your organization&apos;s identity provider</p>
        </div>
      </div>
    </div>
  )
}
