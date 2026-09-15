import { useState } from 'react'
import { LoaderCircle, LogOut, ShieldAlert, TriangleAlert } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'
import { useTranslation } from '../lib/i18n.jsx'

// Shown full-screen when the portal's /auth/me says mustChangePassword -
// blocks access to every module (except app-maintenance itself, see
// verify() in the backend) until the user sets a new password.
export default function ChangePassword({ onChanged, onLogout, onCancel, forced = true }) {
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    if (newPassword !== confirm) {
      setError(t('changePassword.passwordsDontMatch'))
      return
    }
    setLoading(true)
    try {
      const res = await fetch('/app-maintenance/api/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ currentPassword, newPassword }),
      })
      const body = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`)
      onChanged(body)
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
            <h1 className="text-xl font-semibold tracking-tight">{t('changePassword.title')}</h1>
            <p className="text-sm text-muted-foreground">
              {forced ? t('changePassword.expiredNotice') : t('changePassword.voluntaryNotice')}
            </p>
          </div>
        </div>

        <Card className="rounded-2xl shadow-xl shadow-black/5">
          <form onSubmit={submit}>
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                {forced && <ShieldAlert className="h-4 w-4 text-amber-500" />}
                {forced ? t('changePassword.requiredHeading') : t('changePassword.changeHeading')}
              </CardTitle>
              {forced && <CardDescription>{t('changePassword.requiredDescription')}</CardDescription>}
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {error && (
                <Alert variant="destructive">
                  <TriangleAlert />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="grid gap-2">
                <Label htmlFor="current-password">{t('changePassword.currentPassword')}</Label>
                <Input id="current-password" type="password" autoFocus required
                  value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="new-password">{t('changePassword.newPassword')}</Label>
                <Input id="new-password" type="password" required minLength={6}
                  value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="confirm-password">{t('changePassword.confirmPassword')}</Label>
                <Input id="confirm-password" type="password" required minLength={6}
                  value={confirm} onChange={(e) => setConfirm(e.target.value)} />
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-2">
              <Button type="submit" className="w-full" disabled={loading}>
                {loading && <LoaderCircle className="animate-spin" />}
                {t('changePassword.submit')}
              </Button>
              {forced ? (
                <Button type="button" variant="ghost" className="w-full" onClick={onLogout}>
                  <LogOut />
                  {t('changePassword.signOutInstead')}
                </Button>
              ) : (
                <Button type="button" variant="ghost" className="w-full" onClick={onCancel}>
                  {t('common.cancel')}
                </Button>
              )}
            </CardFooter>
          </form>
        </Card>
      </div>
    </div>
  )
}
