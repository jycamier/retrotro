import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronDown, Shield, User as UserIcon, ExternalLink } from 'lucide-react'
import clsx from 'clsx'
import { authApi } from '../../api/client'
import { useAuthStore } from '../../store/authStore'
import type { DevUser, DevUsersResponse } from '../../types'

const roleConfig = {
  admin: {
    badge: 'bg-violet-500 text-white',
    icon: Shield,
    label: 'Admin',
  },
  member: {
    badge: 'bg-gray-500 text-white',
    icon: UserIcon,
    label: 'Membre',
  },
}

export default function DevUserSwitcher() {
  const { user, setAuth } = useAuthStore()
  const navigate = useNavigate()
  const [isOpen, setIsOpen] = useState(false)
  const [devUsers, setDevUsers] = useState<DevUsersResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    authApi.getDevUsers()
      .then(setDevUsers)
      .catch(err => console.error('Failed to fetch dev users:', err))
  }, [])

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSwitch = async (targetUser: DevUser) => {
    if (targetUser.email === user?.email) {
      setIsOpen(false)
      return
    }

    setLoading(true)
    try {
      const result = await authApi.devLogin(targetUser.email, targetUser.displayName)
      setAuth(result.user, result.accessToken)
      setIsOpen(false)
      navigate('/')
    } catch (err) {
      console.error('Failed to switch user:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleOpenNewSession = () => {
    // Generate a unique session ID
    const sessionId = `session-${Date.now()}`
    const url = new URL(window.location.origin)
    url.pathname = '/login'
    url.searchParams.set('session', sessionId)
    window.open(url.toString(), '_blank')
    setIsOpen(false)
  }

  if (!devUsers) return null

  // Find current user's role in the team
  const currentDevUser = devUsers.users.find(u => u.email === user?.email)
  const currentRoleConfig = currentDevUser ? roleConfig[currentDevUser.teamRole] : null
  const CurrentRoleIcon = currentRoleConfig?.icon

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={clsx(
          'flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-colors text-sm',
          'bg-yellow-50 border-yellow-200 hover:bg-yellow-100'
        )}
      >
        <span className="font-medium text-yellow-800">DEV</span>
        {currentRoleConfig && CurrentRoleIcon && (
          <span className={clsx('px-1.5 py-0.5 text-xs font-medium rounded flex items-center gap-1', currentRoleConfig.badge)}>
            <CurrentRoleIcon className="w-3 h-3" />
            {currentRoleConfig.label}
          </span>
        )}
        <ChevronDown className={clsx('w-4 h-4 transition-transform text-yellow-700', isOpen && 'rotate-180')} />
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-72 bg-white rounded-lg shadow-lg border border-gray-200 z-50 overflow-hidden">
          <div className="p-2 bg-gray-50 border-b border-gray-200">
            <span className="text-xs font-medium text-gray-500 uppercase">Switch User</span>
          </div>

          <div className="max-h-80 overflow-y-auto">
            {devUsers.users.map((devUser) => {
              const config = roleConfig[devUser.teamRole]
              const Icon = config.icon
              const isCurrentUser = devUser.email === user?.email

              return (
                <button
                  key={devUser.id}
                  onClick={() => handleSwitch(devUser)}
                  disabled={loading}
                  className={clsx(
                    'w-full px-3 py-2 text-left hover:bg-gray-50 transition-colors flex items-center gap-3',
                    isCurrentUser && 'bg-primary-50',
                    loading && 'opacity-50 cursor-not-allowed'
                  )}
                >
                  <div className={clsx(
                    'w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold',
                    devUser.isAdmin ? 'bg-violet-500 text-white' : 'bg-primary-500 text-white'
                  )}>
                    {devUser.displayName.charAt(0).toUpperCase()}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className={clsx(
                        'font-medium truncate',
                        isCurrentUser ? 'text-primary-700' : 'text-gray-900'
                      )}>
                        {devUser.displayName}
                      </span>
                      {isCurrentUser && (
                        <span className="text-xs text-primary-600">(current)</span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className={clsx('px-1.5 py-0.5 text-xs font-medium rounded flex items-center gap-1', config.badge)}>
                        <Icon className="w-2.5 h-2.5" />
                        {config.label}
                      </span>
                      {devUser.isAdmin && (
                        <span className="px-1.5 py-0.5 text-xs font-medium bg-violet-100 text-violet-700 rounded">
                          Global Admin
                        </span>
                      )}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>

          <div className="p-2 bg-gray-50 border-t border-gray-200">
            <button
              onClick={handleOpenNewSession}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <ExternalLink className="w-4 h-4" />
              Open New Session (new tab)
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
