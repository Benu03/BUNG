import { useState } from 'react'
import { ArrowLeft, Boxes, LogOut, Shield, Users as UsersIcon, LayoutGrid, Settings as SettingsIcon, ScrollText } from 'lucide-react'
import Users from './components/Users.jsx'
import Roles from './components/Roles.jsx'
import Modules from './components/Modules.jsx'
import Settings from './components/Settings.jsx'
import AuditLog from './components/AuditLog.jsx'
import { Button } from './components/ui/button.jsx'
import { cn } from './lib/utils.js'

const TABS = [
  { key: 'users', label: 'Users', icon: UsersIcon, render: () => <Users /> },
  { key: 'roles', label: 'Roles', icon: Shield, render: () => <Roles /> },
  { key: 'modules', label: 'Modules', icon: LayoutGrid, render: () => <Modules /> },
  { key: 'settings', label: 'Settings', icon: SettingsIcon, render: () => <Settings /> },
  { key: 'audit', label: 'Audit Log', icon: ScrollText, render: () => <AuditLog /> },
]

export default function App() {
  const [active, setActive] = useState('users')
  const current = TABS.find((t) => t.key === active)

  const logout = async () => {
    await fetch(`${import.meta.env.BASE_URL}api/auth/logout`, { method: 'POST' })
    window.location.href = '/'
  }

  return (
    <div className="min-h-svh bg-muted/30">
      <header className="sticky top-0 z-10 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3.5">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/15">
              <Boxes className="h-4 w-4" />
            </div>
            <div>
              <div className="text-sm font-semibold leading-none">App Maintenance</div>
              <div className="text-xs text-primary-foreground/70">Users, roles &amp; modules</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" asChild className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <a href="/">
                <ArrowLeft />
                Portal
              </a>
            </Button>
            <Button size="sm" onClick={logout} className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <LogOut />
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">
        <nav className="mb-6 flex gap-1 border-b">
          {TABS.map((t) => {
            const Icon = t.icon
            return (
              <button
                key={t.key}
                onClick={() => setActive(t.key)}
                className={cn(
                  'flex items-center gap-1.5 border-b-2 border-transparent px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground',
                  t.key === active && 'border-primary text-foreground'
                )}
              >
                <Icon className="h-4 w-4" />
                {t.label}
              </button>
            )
          })}
        </nav>

        {current.render()}
      </main>
    </div>
  )
}
