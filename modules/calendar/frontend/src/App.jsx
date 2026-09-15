import { useEffect, useState } from 'react'
import { ArrowLeft, CalendarDays, LogOut, SlidersHorizontal } from 'lucide-react'
import EventsPage from './components/EventsPage.jsx'
import Logo from './components/Logo.jsx'
import SettingsModal from './components/SettingsModal.jsx'
import NotificationBell from './components/NotificationBell.jsx'
import ChatButton from './components/ChatButton.jsx'
import CommandPalette from './components/CommandPalette.jsx'
import { Button } from './components/ui/button.jsx'
import { api } from './api.js'
import { useTranslation } from './lib/i18n.jsx'

export default function App() {
  const { t } = useTranslation()
  const [me, setMe] = useState(null)
  const [showSettings, setShowSettings] = useState(false)

  useEffect(() => { api.me().then(setMe).catch(() => {}) }, [])

  // Auth (login/logout/session) is owned entirely by app-maintenance (see
  // /nginx/auth-common.conf) - this module never implements its own.
  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    window.location.href = '/'
  }

  return (
    <div className="min-h-svh bg-muted/30">
      <header className="sticky top-0 z-10 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3.5">
          <div className="flex items-center gap-2.5">
            <Logo className="h-8 w-8" />
            <div>
              <div className="flex items-center gap-1.5 text-sm font-semibold leading-none">
                <CalendarDays className="h-3.5 w-3.5" />
                {t('header.title')}
              </div>
              <div className="text-xs text-primary-foreground/70">{t('header.subtitle')}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={() => setShowSettings(true)} title={t('common.settings')} className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground">
              <SlidersHorizontal className="h-4 w-4" />
            </Button>
            <CommandPalette className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <NotificationBell className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <ChatButton className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
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

      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} />}

      <main className="mx-auto max-w-4xl px-4 py-8">
        <EventsPage me={me} />
      </main>
    </div>
  )
}
