import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../../store/authStore'
import { LogOut, LayoutDashboard, Users, UsersRound } from 'lucide-react'
import clsx from 'clsx'
import DevUserSwitcher from '../dev/DevUserSwitcher'
import { useState, useEffect } from 'react'
import { authApi } from '../../api/client'

export default function Layout() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()
  const [devMode, setDevMode] = useState(false)

  useEffect(() => {
    authApi.getLoginInfo()
      .then(info => setDevMode(info.devMode))
      .catch(() => setDevMode(false))
  }, [])

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/'
    }
    return location.pathname.startsWith(path)
  }

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-4 py-3">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-8">
            <Link to="/" className="text-xl font-bold text-primary-600">
              RetroTro
            </Link>
            <nav className="flex items-center gap-1">
              <Link
                to="/"
                className={clsx(
                  'flex items-center gap-2 px-3 py-2 text-sm rounded-md transition-colors',
                  isActive('/') && !isActive('/users') && !isActive('/teams-admin')
                    ? 'text-primary-600 bg-primary-50'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                )}
              >
                <LayoutDashboard className="w-4 h-4" />
                Dashboard
              </Link>
              {user?.isAdmin && (
                <>
                  <Link
                    to="/users"
                    className={clsx(
                      'flex items-center gap-2 px-3 py-2 text-sm rounded-md transition-colors',
                      isActive('/users')
                        ? 'text-primary-600 bg-primary-50'
                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                    )}
                  >
                    <Users className="w-4 h-4" />
                    Utilisateurs
                  </Link>
                  <Link
                    to="/teams-admin"
                    className={clsx(
                      'flex items-center gap-2 px-3 py-2 text-sm rounded-md transition-colors',
                      isActive('/teams-admin')
                        ? 'text-primary-600 bg-primary-50'
                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                    )}
                  >
                    <UsersRound className="w-4 h-4" />
                    Équipes
                  </Link>
                </>
              )}
            </nav>
          </div>

          <div className="flex items-center gap-4">
            {devMode && <DevUserSwitcher />}
            {user && (
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  {user.avatarUrl ? (
                    <img
                      src={user.avatarUrl}
                      alt={user.displayName}
                      className="w-8 h-8 rounded-full"
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center">
                      <span className="text-primary-600 font-medium text-sm">
                        {user.displayName.charAt(0).toUpperCase()}
                      </span>
                    </div>
                  )}
                  <span className="text-sm text-gray-700">{user.displayName}</span>
                  {user.isAdmin && (
                    <span className="px-1.5 py-0.5 text-xs font-medium bg-violet-100 text-violet-700 rounded">
                      Admin
                    </span>
                  )}
                </div>
                <button
                  onClick={handleLogout}
                  className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
