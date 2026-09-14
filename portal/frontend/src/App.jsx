import { useEffect, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import { LoaderCircle } from 'lucide-react'

export default function App() {
  const [user, setUser] = useState(undefined) // undefined = loading, null = logged out
  const [notice, setNotice] = useState('')

  const loadMe = () => {
    fetch('/app-maintenance/api/auth/me')
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then(setUser)
      .catch(() => setUser(null))
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('login_required')) setNotice('Please sign in to continue.')
    loadMe()
  }, [])

  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    setUser(null)
  }

  if (user === undefined) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-muted/40">
        <LoaderCircle className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!user) {
    return <Login notice={notice} onLoggedIn={setUser} />
  }

  return <Dashboard user={user} onLogout={logout} />
}
