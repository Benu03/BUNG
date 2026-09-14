import { useEffect, useState } from 'react'
import { KeyRound, Plus, ShieldCheck, Trash2, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'
import { cn } from '../lib/utils.js'

export default function Users() {
  const [users, setUsers] = useState([])
  const [roles, setRoles] = useState([])
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ username: '', fullName: '', email: '', password: '' })
  const [panel, setPanel] = useState(null) // { userId, mode: 'roles' | 'password' }
  const [assignModuleId, setAssignModuleId] = useState('')
  const [passwordInput, setPasswordInput] = useState('')

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
    if (!confirm('Delete this user?')) return
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
          <CardTitle className="text-base">Add user</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5 lg:items-end">
            <div className="grid gap-1.5">
              <Label htmlFor="u-username">Username</Label>
              <Input id="u-username" required value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-fullname">Full name</Label>
              <Input id="u-fullname" required value={form.fullName}
                onChange={(e) => setForm({ ...form, fullName: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-email">Email</Label>
              <Input id="u-email" type="email" required value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="u-password">Password</Label>
              <Input id="u-password" type="password" required minLength={6} placeholder="min 6 chars"
                value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </div>
            <Button type="submit">
              <Plus />
              Add user
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">Username</th>
                <th className="px-4 py-3 font-medium">Full name</th>
                <th className="px-4 py-3 font-medium">Email</th>
                <th className="px-4 py-3 font-medium">Roles</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {users.map((u) => {
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
                        {u.roleIds.map((id) => (
                          <Badge key={id} variant="secondary" title={moduleName(roles.find((r) => r.id === id)?.moduleId)}>
                            {roleName(id)}
                          </Badge>
                        ))}
                        {u.roleIds.length === 0 && <span className="text-xs text-muted-foreground">No roles</span>}
                      </div>

                      {isRolesOpen && (
                        <div className="mt-2 flex flex-col gap-2 rounded-md border bg-muted/30 p-2.5">
                          <div className="grid gap-1">
                            <Label htmlFor="assign-module" className="text-xs text-muted-foreground">1. Pick a module</Label>
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
                            <span className="text-xs text-muted-foreground">2. Toggle role(s) in this module</span>
                            <div className="flex flex-wrap gap-2">
                              {roleIdsInModule.length === 0 && (
                                <span className="text-xs text-muted-foreground">No roles defined for this module yet.</span>
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
                            <span className="text-[11px] text-muted-foreground">A user can hold more than one role within the same module.</span>
                          </div>
                        </div>
                      )}

                      {isPasswordOpen && (
                        <div className="mt-2 flex items-center gap-2">
                          <Input
                            type="password"
                            autoFocus
                            placeholder="New password"
                            className="h-8 max-w-48"
                            value={passwordInput}
                            onChange={(e) => setPasswordInput(e.target.value)}
                          />
                          <Button size="sm" onClick={() => submitPassword(u.id)}>Save</Button>
                          <Button size="sm" variant="ghost" onClick={() => { setPanel(null); setPasswordInput('') }}>Cancel</Button>
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost" size="icon"
                          title="Assign roles"
                          onClick={() => (isRolesOpen ? setPanel(null) : openAssign(u.id))}
                        >
                          <ShieldCheck className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost" size="icon"
                          title="Set password"
                          onClick={() => { setPanel(isPasswordOpen ? null : { userId: u.id, mode: 'password' }); setPasswordInput('') }}
                        >
                          <KeyRound className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" title="Delete" onClick={() => remove(u.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                )
              })}
              {users.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">No users yet.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
