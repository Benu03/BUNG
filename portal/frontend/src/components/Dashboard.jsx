import { useState } from 'react'
import { ArrowRight, Megaphone, PackageOpen } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'
import ThemeToggle from './ThemeToggle.jsx'
import LanguageToggle from './LanguageToggle.jsx'
import ProfileModal from './ProfileModal.jsx'
import { useTranslation } from '../lib/i18n.jsx'

// Pastel duo per module, cycling - soft enough to sit quietly behind dark
// text (see the icon markup below, which uses a dark slate text color
// instead of white for exactly that reason).
const ICON_GRADIENTS = [
  'from-violet-200 to-fuchsia-200',
  'from-sky-200 to-cyan-100',
  'from-amber-100 to-orange-200',
  'from-emerald-200 to-teal-100',
  'from-rose-200 to-pink-200',
  'from-indigo-200 to-blue-100',
]

function initials(name) {
  return name
    .split(' ')
    .map((w) => w[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

export default function Dashboard({ user, onLogout, settings, onChangePassword }) {
  const { t } = useTranslation()
  const [showProfile, setShowProfile] = useState(false)
  const modules = user.modules || []

  return (
    <div className="min-h-svh bg-muted/30">
      <header className="sticky top-0 z-10 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3.5">
          <div className="flex items-center gap-2.5">
            <Logo className="h-8 w-8" />
            <span className="font-semibold tracking-tight">{settings.siteName}</span>
          </div>
          <div className="flex items-center gap-2">
            <LanguageToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <ThemeToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
            <button
              onClick={() => setShowProfile(true)}
              className="flex items-center gap-2 rounded-full border border-white/25 bg-white/10 py-1 pl-1 pr-3 transition-colors hover:bg-white/20"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white/25 text-xs font-semibold">
                {initials(user.fullName || user.username)}
              </span>
              <span className="text-sm">{user.fullName || user.username}</span>
            </button>
          </div>
        </div>
      </header>

      {showProfile && (
        <ProfileModal
          user={user}
          onClose={() => setShowProfile(false)}
          onChangePassword={onChangePassword}
          onLogout={onLogout}
        />
      )}

      <main className="mx-auto max-w-5xl px-4 py-10">
        {settings.announcement && (
          <Alert className="mb-6">
            <Megaphone />
            <AlertDescription>{settings.announcement}</AlertDescription>
          </Alert>
        )}

        <div className="mb-8">
          <h1 className="text-2xl font-semibold tracking-tight">{t('dashboard.yourModules')}</h1>
          <p className="text-sm text-muted-foreground">{t('dashboard.pickModule')}</p>
        </div>

        {modules.length === 0 ? (
          <Card className="flex flex-col items-center gap-2 rounded-2xl border-dashed py-16 text-center">
            <PackageOpen className="h-8 w-8 text-muted-foreground" />
            <p className="font-medium">{t('dashboard.noModulesAssigned')}</p>
            <p className="max-w-xs text-sm text-muted-foreground">{t('dashboard.askAdmin')}</p>
          </Card>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {modules.map((m, i) => (
              <a key={m.code} href={`/${m.code}/`} className="group">
                <Card className="h-full rounded-2xl border-border/60 shadow-sm transition-all duration-200 group-hover:-translate-y-0.5 group-hover:shadow-lg">
                  <CardHeader className="gap-3">
                    <div className={`flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br text-sm font-semibold text-slate-700 ${ICON_GRADIENTS[i % ICON_GRADIENTS.length]}`}>
                      {initials(m.name)}
                    </div>
                    <div>
                      <CardTitle className="flex items-center justify-between text-base">
                        {m.name}
                        <ArrowRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground" />
                      </CardTitle>
                      {m.description && <CardDescription className="mt-1">{m.description}</CardDescription>}
                    </div>
                  </CardHeader>
                  <CardFooter>
                    <span className="text-xs text-muted-foreground">/{m.code}/</span>
                  </CardFooter>
                </Card>
              </a>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
