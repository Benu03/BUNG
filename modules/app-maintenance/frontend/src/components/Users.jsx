import { useEffect, useState } from 'react'
import { KeyRound, Plus, Search, ShieldCheck, Trash2, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'
import { cn } from '../lib/utils.js'
import { useTranslation } from '../lib/i18n.jsx'

// Every seeded role is literally named "Administrator" (one per module),
// so a plain role-name badge just repeats "Administrator" N times with no
// way to tell which module is which at a glance - a pastel color per
// module (cycling, same idea as the portal dashboard's module icons) plus
// leading with the module's name fixes that; the role name only tags
// along when it's something other than the generic default.
const MODULE_BADGE_STYLES = [
  'bg-violet-100 text-violet-700',
  'bg-sky-100 text-sky-700',
  'bg-amber-100 text-amber-700',
  'bg-emerald-100 text-emerald-700',
  'bg-rose-100 text-rose-700',
  'bg-indigo-100 text-indigo-700',
]

export default function Users() {
  const { t } = useTranslation()
  const [users, setUsers] = useState([])
  const [roles, setRoles] = useState([])
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ username: '', fullName: '', email: '', password: '' })
  const [panel, setPanel] = useState(null) // { userId, mode: 'roles' | 'password' }
  const [assignModuleId, setAssignModuleId] = useState('')
  const [passwordInput, setPasswordInput] = useState('')
  const [filter, setFilter] = useState('')

  const load = () => Promise.all([api.listUsers(), api.listRoles(), api.listModules()])
    .then(([u, r, m]) => { setUsers(u); setRoles(r); setModules(m) })
    .catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await api.createUser({ ...form, isActive: true })
      setForm({ username: '', fullName: '', email: '', password: '' })
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (id) => {
    if (!confirm(t('users.confirmDelete'))) return
    await api.deleteUser(id)
    load()
  }

  const toggleRole = async (user, roleId) => {
    const next = user.roleIds.includes(roleId)
      ? user.roleIds.filter((id) => id !== roleId)
      : [...user.roleIds, roleId]
    await api.setUserRoles(user.id, next)
    load()
  }

  const submitPassword = async (userId) => {
    setError('')
    try {
      await api.setUserPassword(userId, passwordInput)
      setPasswordInput('')
      setPanel(null)
    } catch (e) {
      setError(e.message)
    }
  }

  const openAssign = (userId) => {
    setPanel({ userId, mode: 'roles' })
    setAssignModuleId((prev) => prev || modules[0]?.id || '')
  }

  const roleName = (id) => roles.find((r) => r.id === id)?.name || id
  const moduleName = (id) => modules.find((m) => m.id === id)?.name || id
  const rolesForModule = (moduleId) => roles.filter((r) => r.moduleId === moduleId)
  const moduleBadgeStyle = (moduleId) => {
    const idx = modules.findIndex((m) => m.id === moduleId)
    return MODULE_BADGE_STYLES[idx < 0 ? 0 : idx % MODULE_BADGE_STYLES.length]
  }

  const q = filter.trim().toLowerCase()
  const filteredUsers = q
    ? users.filter((u) => [u.username, u.fullName, u.email].some((v) => v?.toLowerCase().includes(q)))
    : users

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
          <CardTitle className="text-base">{t('users.addUser')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5 lg:items-end">
            <div className="grid gap-1.5">
              <Label htmlFor="u-username">{t('users.username')}</Label>
              <Input id="u-username" required value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-fullname">{t('users.fullName')}</Label>
              <Input id="u-fullname" required value={form.fullName}
                onChange={(e) => setForm({ ...form, fullName: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-email">{t('users.email')}</Label>
              <Input id="u-email" type="email" required value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-password">{t('users.password')}</Label>
              <Input id="u-password" type="password" required minLength={6} placeholder={t('users.passwordHint')}
                value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </div>
            <Button type="submit">
              <Plus />
              {t('users.addUserButton')}
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
                <th className="px-4 py-3 font-medium">{t('users.colUsername')}</th>
                <th className="px-4 py-3 font-medium">{t('users.colFullName')}</th>
                <th className="px-4 py-3 font-medium">{t('users.colEmail')}</th>
                <th className="px-4 py-3 font-medium">{t('users.colRoles')}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((u) => {
                const isRolesOpen = panel?.userId === u.id && panel.mode === 'roles'
                const isPasswordOpen = panel?.userId === u.id && panel.mode === 'password'
                const roleIdsInModule = rolesForModule(assignModuleId).map((r) => r.id)
                return (
                  <tr key={u.id} className="border-b last:border-0 align-top hover:bg-muted/40">
                    <td className="px-4 py-3 font-medium">{u.username}</td>
                    <td className="px-4 py-3">{u.fullName}</td>
                    <td className="px-4 py-3 text-muted-foreground">{u.email}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {u.roleIds.map((id) => {
                          const role = roles.find((r) => r.id === id)
                          return (
                            <Badge
                              key={id}
                              variant="outline"
                              className={cn('border-transparent font-medium', moduleBadgeStyle(role?.moduleId))}
                            >
                              {moduleName(role?.moduleId)}
                              {role?.name && role.name !== 'Administrator' && (
                                <span className="ml-1 opacity-70">· {role.name}</span>
                              )}
                            </Badge>
                          )
                        })}
                        {u.roleIds.length === 0 && <span className="text-xs text-muted-foreground">{t('users.noRolesBadge')}</span>}
                      </div>

                      {isRolesOpen && (
                        <div className="mt-2 flex flex-col gap-2 rounded-md border bg-muted/30 p-2.5">
                          <div className="grid gap-1">
                            <Label htmlFor="assign-module" className="text-xs text-muted-foreground">{t('users.pickModuleStep')}</Label>
                            <select
                              id="assign-module"
                              value={assignModuleId}
                              onChange={(e) => setAssignModuleId(e.target.value)}
                              className="h-8 rounded-md border border-input bg-background px-2 text-xs"
                            >
                              {modules.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
                            </select>
                          </div>
                          <div className="grid gap-1">
                            <span className="text-xs text-muted-foreground">{t('users.toggleRoleStep')}</span>
                            <div className="flex flex-wrap gap-2">
                              {roleIdsInModule.length === 0 && (
                                <span className="text-xs text-muted-foreground">{t('users.noRolesForModule')}</span>
                              )}
                              {rolesForModule(assignModuleId).map((r) => {
                                const selected = u.roleIds.includes(r.id)
                                return (
                                  <button
                                    key={r.id}
                                    type="button"
                                    onClick={() => toggleRole(u, r.id)}
                                    className={cn(
                                      'rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                                      selected
                                        ? 'border-primary bg-primary text-primary-foreground'
                                        : 'border-input bg-background hover:bg-accent'
                                    )}
                                  >
                                    {r.name}
                                  </button>
                                )
                              })}
                            </div>
                            <span className="text-[11px] text-muted-foreground">{t('users.multiRoleHint')}</span>
                          </div>
                        </div>
                      )}

                      {isPasswordOpen && (
                        <div className="mt-2 flex items-center gap-2">
                          <Input
                            type="password"
                            autoFocus
                            placeholder={t('users.newPassword')}
                            className="h-8 max-w-48"
                            value={passwordInput}
                            onChange={(e) => setPasswordInput(e.target.value)}
                          />
                          <Button size="sm" onClick={() => submitPassword(u.id)}>{t('users.saveButton')}</Button>
                          <Button size="sm" variant="ghost" onClick={() => { setPanel(null); setPasswordInput('') }}>{t('users.cancelButton')}</Button>
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost" size="icon"
                          title={t('users.assignRoles')}
                          onClick={() => (isRolesOpen ? setPanel(null) : openAssign(u.id))}
                        >
                          <ShieldCheck className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost" size="icon"
                          title={t('users.setPassword')}
                          onClick={() => { setPanel(isPasswordOpen ? null : { userId: u.id, mode: 'password' }); setPasswordInput('') }}
                        >
                          <KeyRound className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" title={t('users.deleteTitle')} onClick={() => remove(u.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                )
              })}
              {filteredUsers.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    {users.length === 0 ? t('users.noUsers') : t('common.noMatches')}
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
