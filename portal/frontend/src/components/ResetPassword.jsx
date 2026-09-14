import { useState } from 'react'
import { CheckCircle2, LoaderCircle, TriangleAlert } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import Logo from './Logo.jsx'

export default function ResetPassword({ token, onDone }) {
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    if (password !== confirm) {
      setError('Passwords do not match.')
      return
    }
    setLoading(true)
    try {
      const res = await fetch('/app-maintenance/api/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, newPassword: password }),
      })
      const body = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`)
      setDone(true)
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
            <h1 className="text-xl font-semibold tracking-tight">Set a new password</h1>
          </div>
        </div>

        <Card className="rounded-2xl shadow-xl shadow-black/5">
          {done ? (
            <CardContent className="flex flex-col items-center gap-3 py-6 text-center">
              <CheckCircle2 className="h-8 w-8 text-emerald-600" />
              <p className="text-sm text-muted-foreground">Your password has been updated.</p>
              <Button onClick={onDone} className="mt-2">Go to sign in</Button>
            </CardContent>
          ) : (
            <form onSubmit={submit}>
              <CardHeader>
                <CardTitle className="text-base">New password</CardTitle>
                <CardDescription>This link is single-use and expires after 1 hour.</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                {error && (
                  <Alert variant="destructive">
                    <TriangleAlert />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <div className="grid gap-2">
                  <Label htmlFor="new-password">New password</Label>
                  <Input id="new-password" type="password" autoFocus required minLength={6}
                    value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="confirm-password">Confirm password</Label>
                  <Input id="confirm-password" type="password" required minLength={6}
                    value={confirm} onChange={(e) => setConfirm(e.target.value)} />
                </div>
              </CardContent>
              <CardFooter>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading && <LoaderCircle className="animate-spin" />}
                  Set new password
                </Button>
              </CardFooter>
            </form>
          )}
        </Card>
      </div>
    </div>
  )
}
