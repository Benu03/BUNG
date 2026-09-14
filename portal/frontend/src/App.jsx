import { useEffect, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import ForgotPassword from './components/ForgotPassword.jsx'
import ResetPassword from './components/ResetPassword.jsx'
import ChangePassword from './components/ChangePassword.jsx'
import { LoaderCircle } from 'lucide-react'

const DEFAULT_SETTINGS = { siteName: 'BUNG', tagline: 'Sign in to access your modules', announcement: '' }

export default function App() {
  const [user, setUser] = useState(undefined) // undefined = loading, null = logged out
  const [notice, setNotice] = useState('')
  const [settings, setSettings] = useState(DEFAULT_SETTINGS)
  const [view, setView] = useState('login') // 'login' | 'forgot' | 'change-password'
  const [resetToken, setResetToken] = useState(null)

  const loadMe = () => {
    fetch('/app-maintenance/api/auth/me')
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then(setUser)
      .catch(() => setUser(null))
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('login_required')) setNotice('Please sign in to continue.')
    const token = params.get('reset_token')
    if (token) setResetToken(token)
    loadMe()
    fetch('/app-maintenance/api/settings/public')
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then(setSettings)
      .catch(() => {}) // keep defaults if app-maintenance isn't reachable yet
  }, [])

  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    setUser(null)
    setView('login')
  }

  const clearResetToken = () => {
    setResetToken(null)
    window.history.replaceState({}, '', '/')
  }

  // A reset link takes over regardless of login state.
  if (resetToken) {
    return <ResetPassword token={resetToken} onDone={clearResetToken} />
  }

  if (user === undefined) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-muted/40">
        <LoaderCircle className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!user) {
    if (view === 'forgot') {
      return <ForgotPassword onBack={() => setView('login')} />
    }
    return <Login notice={notice} onLoggedIn={setUser} settings={settings} onForgotPassword={() => setView('forgot')} />
  }

  if (user.mustChangePassword || view === 'change-password') {
    return (
      <ChangePassword
        forced={user.mustChangePassword}
        onChanged={(updated) => { setUser(updated); setView('login') }}
        onLogout={logout}
        onCancel={() => setView('login')}
      />
    )
  }

  return <Dashboard user={user} onLogout={logout} settings={settings} onChangePassword={() => setView('change-password')} />
}
