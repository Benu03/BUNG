import { KeyRound, LogOut, X } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card, CardContent, CardHeader } from './ui/card.jsx'
import ThemeToggle from './ThemeToggle.jsx'
import LanguageToggle from './LanguageToggle.jsx'
import { useTranslation } from '../lib/i18n.jsx'

function initials(name) {
  return (name || '?')
    .split(' ')
    .map((w) => w[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

export default function ProfileModal({ user, onClose, onChangePassword, onLogout }) {
  const { t } = useTranslation()

  return (
    <div className="fixed inset-0 z-30 flex items-start justify-center bg-black/40 px-4 pt-24" onClick={onClose}>
      <Card className="w-full max-w-xs overflow-hidden rounded-2xl shadow-2xl" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="relative items-center gap-3 border-b bg-gradient-to-br from-primary/10 to-primary-dark/10 pb-6 pt-8 text-center">
          <Button variant="ghost" size="icon" className="absolute right-2 top-2 h-7 w-7" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-primary to-primary-dark text-xl font-semibold text-primary-foreground shadow-md">
            {initials(user.fullName || user.username)}
          </div>
          <div>
            <div className="font-semibold leading-tight">{user.fullName || user.username}</div>
            <div className="text-xs text-muted-foreground">{user.email || `@${user.username}`}</div>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 pt-4">
          <div className="flex items-center justify-between rounded-lg border px-3 py-2">
            <span className="text-sm">{t('common.language')}</span>
            <LanguageToggle />
          </div>
          <div className="flex items-center justify-between rounded-lg border px-3 py-2">
            <span className="text-sm">{t('common.theme')}</span>
            <ThemeToggle />
          </div>
          <Button variant="outline" className="justify-start" onClick={() => { onClose(); onChangePassword() }}>
            <KeyRound />
            {t('dashboard.changePassword')}
          </Button>
          <Button variant="outline" className="justify-start text-destructive hover:text-destructive" onClick={onLogout}>
            <LogOut />
            {t('common.logout')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
