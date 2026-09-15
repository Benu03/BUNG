import { useTranslation } from '../lib/i18n.jsx'
import { Button } from './ui/button.jsx'
import { cn } from '../lib/utils.js'

// Simple two-way toggle, same visual pattern as ThemeToggle - shows the
// language you'd switch TO, not the current one (matches how the theme
// toggle's icon works: it shows the action, not the state).
export default function LanguageToggle({ className }) {
  const { lang, setLang } = useTranslation()

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setLang(lang === 'id' ? 'en' : 'id')}
      title={lang === 'id' ? 'Switch to English' : 'Ganti ke Bahasa Indonesia'}
      className={cn('text-xs font-semibold', className)}
    >
      {lang === 'id' ? 'EN' : 'ID'}
    </Button>
  )
}
