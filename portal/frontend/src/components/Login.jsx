import { useState } from 'react'
import { LoaderCircle, TriangleAlert } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'
import ThemeToggle from './ThemeToggle.jsx'
import LanguageToggle from './LanguageToggle.jsx'
import { useTranslation } from '../lib/i18n.jsx'

export default function Login({ onLoggedIn, notice, settings, onForgotPassword }) {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/app-maintenance/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const body = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(body.error || `Login failed (${res.status})`)
      onLoggedIn(body)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-svh items-center justify-center overflow-hidden bg-background px-4">
      {/* soft ambient glow, brand-colored */}
      <div className="pointer-events-none absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 left-1/2 h-96 w-[36rem] -translate-x-1/2 rounded-full bg-primary/20 blur-3xl" />
        <div className="absolute -bottom-40 right-1/4 h-72 w-72 rounded-full bg-primary-dark/15 blur-3xl" />
      </div>

      <div className="absolute right-4 top-4 z-10 flex items-center gap-1">
        <LanguageToggle className="text-muted-foreground" />
        <ThemeToggle className="text-muted-foreground" />
      </div>

      <div className="relative z-10 w-full max-w-sm">
        <div className="mb-7 flex flex-col items-center gap-3 text-center">
          <Logo className="h-12 w-12 shadow-lg shadow-primary/30" />
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{settings.siteName}</h1>
            <p className="text-sm text-muted-foreground">{settings.tagline}</p>
          </div>
        </div>

        <Card className="rounded-2xl border-border/60 shadow-xl shadow-black/5 backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-base">{t('login.signIn')}</CardTitle>
            <CardDescription>{t('login.enterCredentials')}</CardDescription>
          </CardHeader>
          <form onSubmit={submit}>
            <CardContent className="flex flex-col gap-4">
              {notice && (
                <Alert>
                  <TriangleAlert />
                  <AlertDescription>{notice}</AlertDescription>
                </Alert>
              )}
              {error && (
                <Alert variant="destructive">
                  <TriangleAlert />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="grid gap-2">
                <Label htmlFor="username">{t('login.username')}</Label>
                <Input
                  id="username"
                  autoFocus
                  autoComplete="username"
                  className="h-10 rounded-lg"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                />
              </div>
              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="password">{t('login.password')}</Label>
                  <button type="button" onClick={onForgotPassword} className="text-xs text-muted-foreground hover:text-foreground hover:underline">
                    {t('login.forgotPassword')}
                  </button>
                </div>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  className="h-10 rounded-lg"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-3">
              <Button type="submit" className="h-10 w-full rounded-lg bg-gradient-to-r from-primary to-primary-dark shadow-md shadow-primary/20 hover:opacity-95" disabled={loading}>
                {loading && <LoaderCircle className="animate-spin" />}
                {t('login.signInButton')}
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                {t('login.defaultAdmin')}: <span className="font-mono">admin</span> / <span className="font-mono">admin123</span>
              </p>
            </CardFooter>
          </form>
        </Card>
      </div>
    </div>
  )
}
