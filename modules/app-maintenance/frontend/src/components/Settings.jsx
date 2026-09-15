import { useEffect, useState } from 'react'
import { Save, TriangleAlert, CheckCircle2, ShieldAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'

export default function Settings() {
  const { t } = useTranslation()
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
          <CardTitle className="text-base">{t('settings.portalContent')}</CardTitle>
          <CardDescription>{t('settings.portalContentDesc')}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="s-name">{t('settings.siteName')}</Label>
              <Input id="s-name" required value={form.siteName}
                onChange={(e) => setForm({ ...form, siteName: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="s-tagline">{t('settings.tagline')}</Label>
              <Input id="s-tagline" placeholder={t('settings.taglinePlaceholder')}
                value={form.tagline} onChange={(e) => setForm({ ...form, tagline: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="s-announcement">{t('settings.announcement')}</Label>
              <Input id="s-announcement" placeholder={t('settings.announcementPlaceholder')}
                value={form.announcement} onChange={(e) => setForm({ ...form, announcement: e.target.value })} />
            </div>

            <div className="mt-2 flex items-center gap-1.5 border-t pt-4 text-sm font-medium">
              <ShieldAlert className="h-4 w-4" />
              {t('settings.securityPolicy')}
            </div>
            <div className="grid gap-1.5 sm:max-w-xs">
              <Label htmlFor="s-expiry">{t('settings.passwordExpiry')}</Label>
              <Input id="s-expiry" type="number" min={0} value={form.passwordExpiryDays}
                onChange={(e) => setForm({ ...form, passwordExpiryDays: e.target.value })} />
              <p className="text-xs text-muted-foreground">{t('settings.passwordExpiryHint')}</p>
            </div>

            <div className="flex items-center gap-3">
              <Button type="submit">
                <Save />
                {t('common.save')}
              </Button>
              {saved && (
                <span className="flex items-center gap-1 text-sm text-muted-foreground">
                  <CheckCircle2 className="h-4 w-4 text-emerald-600" />
                  {t('common.saved')}
                </span>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
