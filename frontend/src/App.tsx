import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/authStore'
import Layout from './components/common/Layout'
import LoginPage from './pages/LoginPage'
import CallbackPage from './pages/CallbackPage'
import DashboardPage from './pages/DashboardPage'
import TeamPage from './pages/TeamPage'
import TeamStatsPage from './pages/TeamStatsPage'
import TeamActionsPage from './pages/TeamActionsPage'
import RetroPage from './pages/RetroPage'
import RetroBoardPage from './pages/RetroBoardPage'
import LeanCoffeeBoardPage from './pages/LeanCoffeeBoardPage'
import UsersPage from './pages/UsersPage'
import TeamsAdminPage from './pages/TeamsAdminPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<CallbackPage />} />
      <Route path="/auth/success" element={<CallbackPage />} />

      <Route path="/" element={
        <ProtectedRoute>
          <Layout />
        </ProtectedRoute>
      }>
        <Route index element={<DashboardPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="teams-admin" element={<TeamsAdminPage />} />
        <Route path="teams/:teamId" element={<TeamPage />} />
        <Route path="teams/:teamId/stats" element={<TeamStatsPage />} />
        <Route path="teams/:teamId/actions" element={<TeamActionsPage />} />
        <Route path="teams/:teamId/retros/:retroId" element={<RetroPage />} />
      </Route>

      <Route path="/retro/:retroId" element={
        <ProtectedRoute>
          <RetroBoardPage />
        </ProtectedRoute>
      } />
    </Routes>
  )
}

export default App
