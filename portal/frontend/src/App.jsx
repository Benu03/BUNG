import { useEffect, useState } from 'react'

// Shown if the app-maintenance API can't be reached, so the portal still
// links somewhere useful instead of a blank page.
const FALLBACK_MODULES = [
  { code: 'app-maintenance', name: 'App Maintenance', description: 'Manage users, roles, and modules.' },
  { code: 'kanban', name: 'Kanban', description: 'Boards, columns and cards.' },
]

export default function App() {
  const [modules, setModules] = useState(null)
  const [usedFallback, setUsedFallback] = useState(false)

  useEffect(() => {
    fetch('/app-maintenance/api/modules')
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json()
      })
      .then((data) => setModules(data.filter((m) => m.isActive)))
      .catch(() => {
        setModules(FALLBACK_MODULES)
        setUsedFallback(true)
      })
  }, [])

  return (
    <div className="app">
      <header>
        <h1>BUNG</h1>
        <p>Pick a module to get started.</p>
      </header>

      {usedFallback && (
        <div className="notice">
          Couldn't reach the app-maintenance module registry, showing a fallback list.
        </div>
      )}

      {modules === null ? (
        <div className="muted">Loading modules...</div>
      ) : (
        <div className="module-grid">
          {modules.map((m) => (
            <a key={m.code} className="module-card" href={`/${m.code}/`}>
              <div className="module-card-name">{m.name}</div>
              {m.description && <div className="module-card-desc">{m.description}</div>}
            </a>
          ))}
          {modules.length === 0 && (
            <div className="muted">No active modules yet. Add one via App Maintenance.</div>
          )}
        </div>
      )}
    </div>
  )
}
