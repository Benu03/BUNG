import { X } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card, CardContent, CardHeader } from './ui/card.jsx'
import ThemeToggle from './ThemeToggle.jsx'
import LanguageToggle from './LanguageToggle.jsx'
import { useTranslation } from '../lib/i18n.jsx'

// Language + theme used to be two separate icon buttons directly in the
// header - moved behind one "Settings" icon instead to declutter it
// (requested 2026-09-15). Portal has the fuller ProfileModal instead,
// which folds these two rows in alongside actual profile info; every
// other frontend doesn't have a profile concept of its own, so this
// smaller modal is duplicated into each one's header instead, same
// pattern as NotificationBell/ChatButton.
export default function SettingsModal({ onClose }) {
  const { t } = useTranslation()

  return (
    <div className="fixed inset-0 z-30 flex items-start justify-center bg-black/40 px-4 pt-24" onClick={onClose}>
      <Card className="w-full max-w-xs overflow-hidden rounded-2xl shadow-2xl" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex-row items-center justify-between space-y-0 border-b px-4 py-3">
          <span className="text-sm font-semibold">{t('common.settings')}</span>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
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
        </CardContent>
      </Card>
    </div>
  )
}
