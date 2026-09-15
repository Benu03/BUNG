import { useEffect, useState } from 'react'
import { ArrowLeft, LogOut, MessageCircle } from 'lucide-react'
import ChatPage from './components/ChatPage.jsx'
import Logo from './components/Logo.jsx'
import ThemeToggle from './components/ThemeToggle.jsx'
import LanguageToggle from './components/LanguageToggle.jsx'
import NotificationBell from './components/NotificationBell.jsx'
import CommandPalette from './components/CommandPalette.jsx'
import { Button } from './components/ui/button.jsx'
import { api } from './api.js'
import { useTranslation } from './lib/i18n.jsx'

export default function App() {
  const { t } = useTranslation()
  const [me, setMe] = useState(null)

  useEffect(() => { api.me().then(setMe).catch(() => {}) }, [])

  // Auth (login/logout/session) is owned entirely by app-maintenance (see
  // /nginx/auth-common.conf) - this module never implements its own.
  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    window.location.href = '/'
  }

  return (
    <div className="flex h-svh flex-col bg-muted/30">
      <header className="sticky top-0 z-20 shrink-0 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2.5">
            <Logo className="h-8 w-8" />
            <div>
              <div className="flex items-center gap-1.5 text-sm font-semibold leading-none">
                <MessageCircle className="h-3.5 w-3.5" />
                {t('header.title')}
              </div>
              <div className="text-xs text-primary-foreground/70">{t('header.subtitle')}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <LanguageToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <ThemeToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <CommandPalette className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
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

      <main className="min-h-0 flex-1">
        <ChatPage me={me} />
      </main>
    </div>
  )
}
