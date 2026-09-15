import { useEffect, useState } from 'react'
import { Plus, Search, Trash2, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'
import { useTranslation } from '../lib/i18n.jsx'

export default function Modules() {
  const { t } = useTranslation()
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ code: '', name: '', description: '' })
  const [filter, setFilter] = useState('')

  const load = () => api.listModules().then(setModules).catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await api.createModule({ ...form, isActive: true })
      setForm({ code: '', name: '', description: '' })
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (id) => {
    if (!confirm(t('modules.confirmDelete'))) return
    await api.deleteModule(id)
    load()
  }

  // Deactivating (vs deleting) is reversible - the module immediately
  // disappears from the portal for everyone at their next login (see
  // app-maintenance's UserModules query), without losing its roles/data.
  const toggleActive = async (m) => {
    await api.updateModule(m.id, { ...m, isActive: !m.isActive })
    load()
  }

  const q = filter.trim().toLowerCase()
  const filteredModules = q
    ? modules.filter((m) => [m.code, m.name, m.description].some((v) => v?.toLowerCase().includes(q)))
    : modules

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('modules.addModule')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_2fr_auto] sm:items-end">
            <div className="grid gap-1.5">
              <Label htmlFor="mod-code">{t('modules.code')}</Label>
              <Input id="mod-code" required placeholder="finance"
                value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="mod-name">{t('modules.name')}</Label>
              <Input id="mod-name" required placeholder="Finance"
                value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="mod-desc">{t('modules.description')}</Label>
              <Input id="mod-desc" placeholder={t('common.optional')}
                value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <Button type="submit">
              <Plus />
              {t('modules.addModuleButton')}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <div className="flex items-center gap-2 border-b p-3">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('common.filterPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="h-8 max-w-xs border-0 shadow-none focus-visible:ring-0"
          />
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">{t('modules.colCode')}</th>
                <th className="px-4 py-3 font-medium">{t('modules.colName')}</th>
                <th className="px-4 py-3 font-medium">{t('modules.colDescription')}</th>
                <th className="px-4 py-3 font-medium">{t('modules.colStatus')}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {filteredModules.map((m) => (
                <tr key={m.id} className="border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3 font-mono text-xs">{m.code}</td>
                  <td className="px-4 py-3 font-medium">{m.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{m.description}</td>
                  <td className="px-4 py-3">
                    <button onClick={() => toggleActive(m)} title={t('modules.toggleActiveHint')}>
                      <Badge variant={m.isActive ? 'default' : 'secondary'} className="cursor-pointer">
                        {m.isActive ? t('modules.active') : t('modules.inactive')}
                      </Badge>
                    </button>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button variant="ghost" size="icon" onClick={() => remove(m.id)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </td>
                </tr>
              ))}
              {filteredModules.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    {modules.length === 0 ? t('modules.noModules') : t('common.noMatches')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
