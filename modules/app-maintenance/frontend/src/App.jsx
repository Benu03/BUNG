import { useState } from 'react'
import { ArrowLeft, LogOut, Shield, Users as UsersIcon, LayoutGrid, Settings as SettingsIcon, ScrollText } from 'lucide-react'
import Users from './components/Users.jsx'
import Roles from './components/Roles.jsx'
import Modules from './components/Modules.jsx'
import Settings from './components/Settings.jsx'
import AuditLog from './components/AuditLog.jsx'
import Logo from './components/Logo.jsx'
import ThemeToggle from './components/ThemeToggle.jsx'
import LanguageToggle from './components/LanguageToggle.jsx'
import NotificationBell from './components/NotificationBell.jsx'
import { Button } from './components/ui/button.jsx'
import { cn } from './lib/utils.js'
import { useTranslation } from './lib/i18n.jsx'

const TABS = [
  { key: 'users', labelKey: 'tabs.users', icon: UsersIcon, render: () => <Users /> },
  { key: 'roles', labelKey: 'tabs.roles', icon: Shield, render: () => <Roles /> },
  { key: 'modules', labelKey: 'tabs.modules', icon: LayoutGrid, render: () => <Modules /> },
  { key: 'settings', labelKey: 'tabs.settings', icon: SettingsIcon, render: () => <Settings /> },
  { key: 'audit', labelKey: 'tabs.auditLog', icon: ScrollText, render: () => <AuditLog /> },
]

export default function App() {
  const { t } = useTranslation()
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
            <Logo className="h-8 w-8" />
            <div>
              <div className="text-sm font-semibold leading-none">{t('header.title')}</div>
              <div className="text-xs text-primary-foreground/70">{t('header.subtitle')}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <LanguageToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <ThemeToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <NotificationBell className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <Button size="sm" asChild className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <a href="/">
                <ArrowLeft />
                {t('common.portal')}
              </a>
            </Button>
            <Button size="sm" onClick={logout} className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <LogOut />
              {t('common.logout')}
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">
        <nav className="mb-6 flex gap-1 border-b">
          {TABS.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.key}
                onClick={() => setActive(tab.key)}
                className={cn(
                  'flex items-center gap-1.5 border-b-2 border-transparent px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground',
                  tab.key === active && 'border-primary text-foreground'
                )}
              >
                <Icon className="h-4 w-4" />
                {t(tab.labelKey)}
              </button>
            )
          })}
        </nav>

        {current.render()}
      </main>
    </div>
  )
}
