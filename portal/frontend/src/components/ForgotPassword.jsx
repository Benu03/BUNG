import { useState } from 'react'
import { ArrowLeft, LoaderCircle, MailCheck, TriangleAlert } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'
import { useTranslation } from '../lib/i18n.jsx'

export default function ForgotPassword({ onBack }) {
  const { t } = useTranslation()
  const [identifier, setIdentifier] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/app-maintenance/api/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ identifier }),
      })
      const body = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`)
      setSent(true)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm">
        <div className="mb-7 flex flex-col items-center gap-3 text-center">
          <Logo className="h-12 w-12" />
          <div>
            <h1 className="text-xl font-semibold tracking-tight">{t('forgotPassword.title')}</h1>
            <p className="text-sm text-muted-foreground">{t('forgotPassword.subtitle')}</p>
          </div>
        </div>

        <Card className="rounded-2xl shadow-xl shadow-black/5">
          <CardHeader>
            <CardTitle className="text-base">{t('forgotPassword.heading')}</CardTitle>
            <CardDescription>{t('forgotPassword.description')}</CardDescription>
          </CardHeader>
          {sent ? (
            <CardContent className="flex flex-col items-center gap-3 py-6 text-center">
              <MailCheck className="h-8 w-8 text-emerald-600" />
              <p className="text-sm text-muted-foreground">{t('forgotPassword.sentMessage')}</p>
              <Button variant="outline" onClick={onBack} className="mt-2">
                <ArrowLeft />
                {t('forgotPassword.backToSignIn')}
              </Button>
            </CardContent>
          ) : (
            <form onSubmit={submit}>
              <CardContent className="flex flex-col gap-4">
                {error && (
                  <Alert variant="destructive">
                    <TriangleAlert />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <div className="grid gap-2">
                  <Label htmlFor="identifier">{t('forgotPassword.usernameOrEmail')}</Label>
                  <Input id="identifier" autoFocus required value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)} />
                </div>
              </CardContent>
              <CardFooter className="flex-col gap-3">
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading && <LoaderCircle className="animate-spin" />}
                  {t('forgotPassword.sendLink')}
                </Button>
                <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
                  <ArrowLeft />
                  {t('forgotPassword.backToSignIn')}
                </Button>
              </CardFooter>
            </form>
          )}
        </Card>
      </div>
    </div>
  )
}
