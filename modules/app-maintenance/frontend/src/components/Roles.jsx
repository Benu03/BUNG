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

export default function Roles() {
  const { t } = useTranslation()
  const [roles, setRoles] = useState([])
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ name: '', description: '', moduleId: '' })
  const [filter, setFilter] = useState('')

  const load = () => Promise.all([api.listRoles(), api.listModules()])
    .then(([r, m]) => {
      setRoles(r)
      setModules(m)
      setForm((f) => ({ ...f, moduleId: f.moduleId || m[0]?.id || '' }))
    })
    .catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await api.createRole(form)
      setForm({ name: '', description: '', moduleId: form.moduleId })
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (id) => {
    if (!confirm(t('roles.confirmDelete'))) return
    await api.deleteRole(id)
    load()
  }

  // Deactivating (vs deleting) is reversible - anyone holding this role
  // immediately stops getting the module access it grants at their next
  // login (see app-maintenance's UserModules query), without losing the
  // role assignment itself.
  const toggleActive = async (role) => {
    await api.updateRole(role.id, { ...role, isActive: !role.isActive })
    load()
  }

  const moduleName = (id) => modules.find((m) => m.id === id)?.name || id

  const q = filter.trim().toLowerCase()
  const filteredRoles = q
    ? roles.filter((r) => [r.name, r.description, moduleName(r.moduleId)].some((v) => v?.toLowerCase().includes(q)))
    : roles

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
          <CardTitle className="text-base">{t('roles.addRole')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-[1fr_1fr_1.4fr_auto] lg:items-end">
            <div className="grid gap-1.5">
              <Label htmlFor="role-module">{t('roles.module')}</Label>
              <select
                id="role-module"
                required
                value={form.moduleId}
                onChange={(e) => setForm({ ...form, moduleId: e.target.value })}
                className="h-9 rounded-md border border-input bg-background px-3 text-sm shadow-sm"
              >
                {modules.length === 0 && <option value="">{t('roles.noModulesYet')}</option>}
                {modules.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="role-name">{t('roles.name')}</Label>
              <Input id="role-name" required placeholder={t('roles.namePlaceholder')}
                value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="role-desc">{t('roles.description')}</Label>
              <Input id="role-desc" placeholder={t('common.optional')}
                value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <Button type="submit" disabled={!form.moduleId}>
              <Plus />
              {t('roles.addRoleButton')}
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
                <th className="px-4 py-3 font-medium">{t('roles.colModule')}</th>
                <th className="px-4 py-3 font-medium">{t('roles.colName')}</th>
                <th className="px-4 py-3 font-medium">{t('roles.colDescription')}</th>
                <th className="px-4 py-3 font-medium">{t('roles.colStatus')}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {filteredRoles.map((r) => (
                <tr key={r.id} className="border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3"><Badge variant="secondary">{moduleName(r.moduleId)}</Badge></td>
                  <td className="px-4 py-3 font-medium">{r.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{r.description}</td>
                  <td className="px-4 py-3">
                    <button onClick={() => toggleActive(r)} title={t('roles.toggleActiveHint')}>
                      <Badge variant={r.isActive ? 'default' : 'secondary'} className="cursor-pointer">
                        {r.isActive ? t('roles.active') : t('roles.inactive')}
                      </Badge>
                    </button>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button variant="ghost" size="icon" onClick={() => remove(r.id)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </td>
                </tr>
              ))}
              {filteredRoles.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    {roles.length === 0 ? t('roles.noRoles') : t('common.noMatches')}
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
