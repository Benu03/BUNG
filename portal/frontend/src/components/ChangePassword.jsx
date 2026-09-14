import { useState } from 'react'
import { LoaderCircle, LogOut, ShieldAlert, TriangleAlert } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'

// Shown full-screen when the portal's /auth/me says mustChangePassword -
// blocks access to every module (except app-maintenance itself, see
// verify() in the backend) until the user sets a new password.
export default function ChangePassword({ onChanged, onLogout, onCancel, forced = true }) {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    if (newPassword !== confirm) {
      setError('Passwords do not match.')
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
            <h1 className="text-xl font-semibold tracking-tight">Update your password</h1>
            <p className="text-sm text-muted-foreground">
              {forced ? 'Your password has expired and needs to be changed.' : 'Choose a new password for your account.'}
            </p>
          </div>
        </div>

        <Card className="rounded-2xl shadow-xl shadow-black/5">
          <form onSubmit={submit}>
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                {forced && <ShieldAlert className="h-4 w-4 text-amber-500" />}
                {forced ? 'Password change required' : 'Change password'}
              </CardTitle>
              {forced && <CardDescription>You can't continue until you set a new password.</CardDescription>}
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {error && (
                <Alert variant="destructive">
                  <TriangleAlert />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="grid gap-2">
                <Label htmlFor="current-password">Current password</Label>
                <Input id="current-password" type="password" autoFocus required
                  value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="new-password">New password</Label>
                <Input id="new-password" type="password" required minLength={6}
                  value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="confirm-password">Confirm new password</Label>
                <Input id="confirm-password" type="password" required minLength={6}
                  value={confirm} onChange={(e) => setConfirm(e.target.value)} />
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-2">
              <Button type="submit" className="w-full" disabled={loading}>
                {loading && <LoaderCircle className="animate-spin" />}
                Update password
              </Button>
              {forced ? (
                <Button type="button" variant="ghost" className="w-full" onClick={onLogout}>
                  <LogOut />
                  Sign out instead
                </Button>
              ) : (
                <Button type="button" variant="ghost" className="w-full" onClick={onCancel}>
                  Cancel
                </Button>
              )}
            </CardFooter>
          </form>
        </Card>
      </div>
    </div>
  )
}
