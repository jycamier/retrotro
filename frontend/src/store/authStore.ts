import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '../types'

// Get session ID from URL for multi-tab isolation
function getSessionId(): string {
  if (typeof window === 'undefined') return 'default'
  const params = new URLSearchParams(window.location.search)
  return params.get('session') || 'default'
}

// Store session ID for use elsewhere
export function getCurrentSessionId(): string {
  return getSessionId()
}

interface AuthState {
  user: User | null
  accessToken: string | null
  isAuthenticated: boolean
  setAuth: (user: User, token: string) => void
  logout: () => void
  updateUser: (user: User) => void
}

const sessionId = getSessionId()
const storageName = sessionId === 'default' ? 'auth-storage' : `auth-storage-${sessionId}`

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      setAuth: (user, token) => set({
        user,
        accessToken: token,
        isAuthenticated: true,
      }),
      logout: () => set({
        user: null,
        accessToken: null,
        isAuthenticated: false,
      }),
      updateUser: (user) => set({ user }),
    }),
    {
      name: storageName,
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
