import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { AgentSessionProvider } from './context/AgentSessionContext'
import { PersonProvider } from './context/PersonContext'
import { ThemeProvider } from './context/ThemeContext'
import App from './App'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <PersonProvider>
          <AgentSessionProvider>
            <ThemeProvider>
              <App />
            </ThemeProvider>
          </AgentSessionProvider>
        </PersonProvider>
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>
)
