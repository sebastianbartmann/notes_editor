import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import Layout from './components/Layout/Layout'
import DailyPage from './pages/DailyPage'
import FilesPage from './pages/FilesPage'
import SleepPage from './pages/SleepPage'
import ClaudePage from './pages/ClaudePage'
import NoisePage from './pages/NoisePage'
import SettingsPage from './pages/SettingsPage'
import LoginPage from './pages/LoginPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/*"
        element={
          <ProtectedRoute>
            <Layout>
              <Routes>
                <Route path="/" element={<Navigate to="/daily" replace />} />
                <Route path="/daily" element={<DailyPage />} />
                <Route path="/files" element={<FilesPage />} />
                <Route path="/files/*" element={<FilesPage />} />
                <Route path="/sleep" element={<SleepPage />} />
                <Route path="/claude" element={<ClaudePage />} />
                <Route path="/noise" element={<NoisePage />} />
                <Route path="/settings" element={<SettingsPage />} />
              </Routes>
            </Layout>
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}

export default App
