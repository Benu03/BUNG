import { useEffect, useState } from 'react'
import { Megaphone, PackageOpen, Search, Star } from 'lucide-react'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'
import ThemeToggle from './ThemeToggle.jsx'
import LanguageToggle from './LanguageToggle.jsx'
import ProfileModal from './ProfileModal.jsx'
import NotificationBell from './NotificationBell.jsx'
import { useTranslation } from '../lib/i18n.jsx'

const FAVORITES_KEY = 'bung-favorite-modules'

// Per-browser only (same as theme/language) - there's no per-user
// preference storage on the backend for this yet, and it doesn't need to
// sync across devices to be useful.
function loadFavorites() {
  try {
    const raw = localStorage.getItem(FAVORITES_KEY)
    return raw ? new Set(JSON.parse(raw)) : new Set()
  } catch {
    return new Set()
  }
}
function saveFavorites(set) {
  try {
    localStorage.setItem(FAVORITES_KEY, JSON.stringify([...set]))
  } catch {
    // ignore (private browsing, storage disabled, etc.)
  }
}

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
  const [filter, setFilter] = useState('')
  const [favorites, setFavorites] = useState(loadFavorites)
  const modules = user.modules || []

  useEffect(() => { saveFavorites(favorites) }, [favorites])

  const toggleFavorite = (code, e) => {
    e.preventDefault()
    e.stopPropagation()
    setFavorites((prev) => {
      const next = new Set(prev)
      if (next.has(code)) next.delete(code)
      else next.add(code)
      return next
    })
  }

  const q = filter.trim().toLowerCase()
  const visibleModules = modules
    .filter((m) => !q || [m.name, m.description, m.code].some((v) => v?.toLowerCase().includes(q)))
    // Favorites first, stable otherwise (Array.prototype.sort is stable
    // in modern JS engines).
    .slice()
    .sort((a, b) => Number(favorites.has(b.code)) - Number(favorites.has(a.code)))

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
            <NotificationBell className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
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

        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('dashboard.yourModules')}</h1>
            <p className="text-sm text-muted-foreground">{t('dashboard.pickModule')}</p>
          </div>
          {modules.length > 0 && (
            <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2">
              <Search className="h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t('common.filterPlaceholder')}
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="h-8 w-48 border-0 shadow-none focus-visible:ring-0"
              />
            </div>
          )}
        </div>

        {modules.length === 0 ? (
          <Card className="flex flex-col items-center gap-2 rounded-2xl border-dashed py-16 text-center">
            <PackageOpen className="h-8 w-8 text-muted-foreground" />
            <p className="font-medium">{t('dashboard.noModulesAssigned')}</p>
            <p className="max-w-xs text-sm text-muted-foreground">{t('dashboard.askAdmin')}</p>
          </Card>
        ) : visibleModules.length === 0 ? (
          <Card className="flex flex-col items-center gap-2 rounded-2xl border-dashed py-16 text-center text-muted-foreground">
            {t('common.noMatches')}
          </Card>
        ) : (
          // Compact icon-tile grid (Odoo app-launcher style) - the icon
          // does the identifying, the name is a one-line label underneath,
          // and the description moves to a hover tooltip instead of taking
          // up its own line, so far more modules fit on screen at once.
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 sm:gap-3 md:grid-cols-5 lg:grid-cols-6">
            {visibleModules.map((m, i) => {
              const isFavorite = favorites.has(m.code)
              return (
                <a key={m.code} href={`/${m.code}/`} className="group" title={m.description}>
                  <div className="relative flex flex-col items-center gap-2 rounded-xl px-2 py-3 text-center transition-colors hover:bg-card hover:shadow-sm">
                    <button
                      onClick={(e) => toggleFavorite(m.code, e)}
                      title={t(isFavorite ? 'dashboard.removeFavorite' : 'dashboard.addFavorite')}
                      className={`absolute right-0.5 top-0.5 rounded-full p-1 transition-opacity ${
                        isFavorite ? 'text-amber-500 opacity-100' : 'text-muted-foreground/40 opacity-0 hover:text-amber-500 group-hover:opacity-100'
                      }`}
                    >
                      <Star className={`h-3.5 w-3.5 ${isFavorite ? 'fill-amber-400' : ''}`} />
                    </button>
                    <div className={`flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br text-base font-semibold text-slate-700 shadow-sm transition-transform duration-200 group-hover:-translate-y-0.5 group-hover:shadow-md sm:h-16 sm:w-16 ${ICON_GRADIENTS[i % ICON_GRADIENTS.length]}`}>
                      {initials(m.name)}
                    </div>
                    <span className="line-clamp-2 text-xs font-medium leading-tight text-foreground sm:text-sm">{m.name}</span>
                  </div>
                </a>
              )
            })}
          </div>
        )}
      </main>
    </div>
  )
}
