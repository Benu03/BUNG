import { useEffect, useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { cn } from '../lib/utils.js'

const STORAGE_KEY = 'bung-theme'

// Default is always "light" (set synchronously in index.html before paint,
// see the inline script there) - this never falls back to the OS/browser
// preference, only to what's in localStorage.
export default function ThemeToggle({ className }) {
  const [theme, setTheme] = useState(() => document.documentElement.getAttribute('data-theme') || 'light')

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))}
      title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
      className={cn(className)}
    >
      {theme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </Button>
  )
}
