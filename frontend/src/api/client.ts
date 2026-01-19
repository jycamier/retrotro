import { useAuthStore } from '../store/authStore'
import type { ApiError } from '../types'

const API_BASE = '/api/v1'

class ApiClient {
  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }

    const token = useAuthStore.getState().accessToken
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    return headers
  }

  async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
    })

    if (response.status === 401) {
      // Try to refresh token
      const refreshed = await this.refreshToken()
      if (refreshed) {
        // Retry request
        return this.request(path, options)
      }
      // Logout if refresh failed
      useAuthStore.getState().logout()
      throw new Error('Unauthorized')
    }

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error)
    }

    if (response.status === 204) {
      return {} as T
    }

    return response.json()
  }

  async refreshToken(): Promise<boolean> {
    try {
      const response = await fetch('/auth/refresh', {
        method: 'POST',
        credentials: 'include',
      })

      if (!response.ok) {
        return false
      }

      const data = await response.json()
      const currentUser = useAuthStore.getState().user
      if (currentUser) {
        useAuthStore.getState().setAuth(currentUser, data.accessToken)
      }
      return true
    } catch {
      return false
    }
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'GET' })
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'DELETE' })
  }
}

export const api = new ApiClient()

// API functions
export const teamsApi = {
  list: () => api.get<Team[]>('/teams'),
  get: (id: string) => api.get<Team>(`/teams/${id}`),
  create: (data: { name: string; slug: string; description?: string }) =>
    api.post<Team>('/teams', data),
  update: (id: string, data: { name?: string; description?: string }) =>
    api.put<Team>(`/teams/${id}`, data),
  delete: (id: string) => api.delete(`/teams/${id}`),
  getMembers: (id: string) => api.get<TeamMember[]>(`/teams/${id}/members`),
  addMember: (teamId: string, userId: string, role: string) =>
    api.post(`/teams/${teamId}/members`, { userId, role }),
  removeMember: (teamId: string, userId: string) =>
    api.delete(`/teams/${teamId}/members/${userId}`),
  updateMemberRole: (teamId: string, userId: string, role: string) =>
    api.put(`/teams/${teamId}/members/${userId}/role`, { role }),
}

export const templatesApi = {
  list: (teamId?: string) => api.get<Template[]>(`/templates${teamId ? `?teamId=${teamId}` : ''}`),
  get: (id: string) => api.get<Template>(`/templates/${id}`),
  create: (data: Partial<Template>) => api.post<Template>('/templates', data),
}

export const retrosApi = {
  list: (teamId: string, status?: string) =>
    api.get<Retrospective[]>(`/retrospectives?teamId=${teamId}${status ? `&status=${status}` : ''}`),
  get: (id: string) => api.get<Retrospective>(`/retrospectives/${id}`),
  create: (data: {
    name: string
    teamId: string
    templateId: string
    maxVotesPerUser?: number
    anonymousVoting?: boolean
  }) => api.post<Retrospective>('/retrospectives', data),
  update: (id: string, data: Partial<Retrospective>) =>
    api.put<Retrospective>(`/retrospectives/${id}`, data),
  delete: (id: string) => api.delete(`/retrospectives/${id}`),
  start: (id: string) => api.post<Retrospective>(`/retrospectives/${id}/start`),
  end: (id: string) => api.post<Retrospective>(`/retrospectives/${id}/end`),
  getItems: (id: string) => api.get<Item[]>(`/retrospectives/${id}/items`),
  createItem: (retroId: string, data: { columnId: string; content: string }) =>
    api.post<Item>(`/retrospectives/${retroId}/items`, data),
  updateItem: (retroId: string, itemId: string, data: { content: string }) =>
    api.put<Item>(`/retrospectives/${retroId}/items/${itemId}`, data),
  deleteItem: (retroId: string, itemId: string) =>
    api.delete(`/retrospectives/${retroId}/items/${itemId}`),
  vote: (retroId: string, itemId: string) =>
    api.post(`/retrospectives/${retroId}/items/${itemId}/vote`),
  unvote: (retroId: string, itemId: string) =>
    api.delete(`/retrospectives/${retroId}/items/${itemId}/vote`),
  getActions: (id: string) => api.get<ActionItem[]>(`/retrospectives/${id}/actions`),
  createAction: (retroId: string, data: Partial<ActionItem>) =>
    api.post<ActionItem>(`/retrospectives/${retroId}/actions`, data),
  updateAction: (retroId: string, actionId: string, data: Partial<ActionItem>) =>
    api.put<ActionItem>(`/retrospectives/${retroId}/actions/${actionId}`, data),
  deleteAction: (retroId: string, actionId: string) =>
    api.delete(`/retrospectives/${retroId}/actions/${actionId}`),
  // Timer
  startTimer: (retroId: string, durationSeconds?: number) =>
    api.post(`/retrospectives/${retroId}/timer/start`, { duration_seconds: durationSeconds }),
  pauseTimer: (retroId: string) =>
    api.post(`/retrospectives/${retroId}/timer/pause`),
  resumeTimer: (retroId: string) =>
    api.post(`/retrospectives/${retroId}/timer/resume`),
  addTime: (retroId: string, seconds: number) =>
    api.post(`/retrospectives/${retroId}/timer/add-time`, { seconds }),
  nextPhase: (retroId: string) =>
    api.post<{ phase: string }>(`/retrospectives/${retroId}/phase/next`),
  setPhase: (retroId: string, phase: string) =>
    api.post(`/retrospectives/${retroId}/phase/set`, { phase }),
  // ROTI and Icebreaker
  getRotiResults: (retroId: string) =>
    api.get<RotiResults>(`/retrospectives/${retroId}/roti`),
  getIcebreakerMoods: (retroId: string) =>
    api.get<IcebreakerMood[]>(`/retrospectives/${retroId}/icebreaker`),
}

export const userApi = {
  me: () => api.get<User>('/me'),
}

export const adminApi = {
  listUsers: () => api.get<User[]>('/admin/users'),
  listTeams: () => api.get<TeamWithMemberCount[]>('/admin/teams'),
  getTeamMembers: (teamId: string) => api.get<TeamMember[]>(`/admin/teams/${teamId}/members`),
}

export const statsApi = {
  getTeamRotiStats: (teamId: string, limit?: number) =>
    api.get<TeamRotiStats>(`/teams/${teamId}/stats/roti${limit ? `?limit=${limit}` : ''}`),
  getTeamMoodStats: (teamId: string, limit?: number) =>
    api.get<TeamMoodStats>(`/teams/${teamId}/stats/mood${limit ? `?limit=${limit}` : ''}`),
  getUserRotiStats: (teamId: string, userId: string, limit?: number) =>
    api.get<UserRotiStats>(`/teams/${teamId}/stats/users/${userId}/roti${limit ? `?limit=${limit}` : ''}`),
  getUserMoodStats: (teamId: string, userId: string, limit?: number) =>
    api.get<UserMoodStats>(`/teams/${teamId}/stats/users/${userId}/mood${limit ? `?limit=${limit}` : ''}`),
  getMyStats: (teamId: string, limit?: number) =>
    api.get<CombinedUserStats>(`/teams/${teamId}/stats/me${limit ? `?limit=${limit}` : ''}`),
}

// Auth API (not using base /api/v1 path)
export const authApi = {
  getLoginInfo: async (): Promise<{ oidcConfigured: boolean; devMode: boolean }> => {
    const response = await fetch('/auth/info')
    return response.json()
  },
  devLogin: async (email: string, displayName: string): Promise<{ user: User; accessToken: string; expiresAt: string }> => {
    const response = await fetch('/auth/dev-login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, displayName }),
      credentials: 'include',
    })
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Login failed' }))
      throw new Error(error.error)
    }
    return response.json()
  },
  logout: async (): Promise<void> => {
    await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
  },
  getDevUsers: async (): Promise<DevUsersResponse> => {
    const response = await fetch('/auth/dev-users')
    if (!response.ok) {
      throw new Error('Failed to fetch dev users')
    }
    return response.json()
  },
}

// Import types
import type { Team, TeamMember, TeamWithMemberCount, Template, Retrospective, Item, ActionItem, User, RotiResults, IcebreakerMood, TeamRotiStats, TeamMoodStats, UserRotiStats, UserMoodStats, CombinedUserStats, DevUsersResponse } from '../types'
