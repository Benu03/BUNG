import { useState } from 'react'
import Users from './components/Users.jsx'
import Roles from './components/Roles.jsx'
import Modules from './components/Modules.jsx'

const TABS = [
  { key: 'users', label: 'Users', render: () => <Users /> },
  { key: 'roles', label: 'Roles', render: () => <Roles /> },
  { key: 'modules', label: 'Modules', render: () => <Modules /> },
]

export default function App() {
  const [active, setActive] = useState('users')
  const current = TABS.find((t) => t.key === active)

  return (
    <div className="app">
      <header>
        <div>
          <h1>App Maintenance</h1>
          <p>Manage users, roles, and the modules they can access.</p>
        </div>
      </header>

      <nav className="tabs">
        {TABS.map((t) => (
          <button
            key={t.key}
            className={t.key === active ? 'active' : ''}
            onClick={() => setActive(t.key)}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {current.render()}
    </div>
  )
}
