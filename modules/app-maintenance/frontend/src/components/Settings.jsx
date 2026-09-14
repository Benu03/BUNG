import { useEffect, useState } from 'react'
import { Save, TriangleAlert, CheckCircle2, ShieldAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'

export default function Settings() {
  const [form, setForm] = useState({ siteName: '', tagline: '', announcement: '', passwordExpiryDays: 60 })
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    api.getSettings().then(setForm).catch((e) => setError(e.message))
  }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setSaved(false)
    try {
      const updated = await api.updateSettings({ ...form, passwordExpiryDays: Number(form.passwordExpiryDays) || 0 })
      setForm(updated)
      setSaved(true)
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle className="text-base">Portal content</CardTitle>
          <CardDescription>Shown on the public landing page (/) before anyone signs in.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="s-name">Site name</Label>
              <Input id="s-name" required value={form.siteName}
                onChange={(e) => setForm({ ...form, siteName: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="s-tagline">Tagline</Label>
              <Input id="s-tagline" placeholder="Shown under the site name on the login card"
                value={form.tagline} onChange={(e) => setForm({ ...form, tagline: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="s-announcement">Announcement</Label>
              <Input id="s-announcement" placeholder="Optional banner shown on the module dashboard - leave blank to hide"
                value={form.announcement} onChange={(e) => setForm({ ...form, announcement: e.target.value })} />
            </div>

            <div className="mt-2 flex items-center gap-1.5 border-t pt-4 text-sm font-medium">
              <ShieldAlert className="h-4 w-4" />
              Security policy
            </div>
            <div className="grid gap-1.5 sm:max-w-xs">
              <Label htmlFor="s-expiry">Password expiry (days)</Label>
              <Input id="s-expiry" type="number" min={0} value={form.passwordExpiryDays}
                onChange={(e) => setForm({ ...form, passwordExpiryDays: e.target.value })} />
              <p className="text-xs text-muted-foreground">
                Users are forced to change their password after this many days (default 60, ~2 months). Set to 0 to disable.
              </p>
            </div>

            <div className="flex items-center gap-3">
              <Button type="submit">
                <Save />
                Save
              </Button>
              {saved && (
                <span className="flex items-center gap-1 text-sm text-muted-foreground">
                  <CheckCircle2 className="h-4 w-4 text-emerald-600" />
                  Saved
                </span>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
