import { useEffect, useState } from 'react'
import { Plus, Trash2, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'
import { cn } from '../lib/utils.js'

export default function Roles() {
  const [roles, setRoles] = useState([])
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ name: '', description: '', moduleIds: [] })

  const load = () => Promise.all([api.listRoles(), api.listModules()])
    .then(([r, m]) => { setRoles(r); setModules(m) })
    .catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const toggleModule = (id) => {
    setForm((f) => ({
      ...f,
      moduleIds: f.moduleIds.includes(id) ? f.moduleIds.filter((x) => x !== id) : [...f.moduleIds, id],
    }))
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await api.createRole(form)
      setForm({ name: '', description: '', moduleIds: [] })
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (id) => {
    if (!confirm('Delete this role?')) return
    await api.deleteRole(id)
    load()
  }

  const moduleName = (id) => modules.find((m) => m.id === id)?.name || id

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
          <CardTitle className="text-base">Add role</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="role-name">Name</Label>
                <Input id="role-name" required placeholder="Administrator"
                  value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="role-desc">Description</Label>
                <Input id="role-desc" placeholder="optional"
                  value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
              </div>
            </div>
            <div className="grid gap-1.5">
              <Label>Modules</Label>
              <div className="flex flex-wrap gap-2">
                {modules.map((m) => {
                  const selected = form.moduleIds.includes(m.id)
                  return (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => toggleModule(m.id)}
                      className={cn(
                        'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
                        selected
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-input bg-background text-foreground hover:bg-accent'
                      )}
                    >
                      {m.name}
                    </button>
                  )
                })}
                {modules.length === 0 && <span className="text-sm text-muted-foreground">No modules yet.</span>}
              </div>
            </div>
            <div>
              <Button type="submit">
                <Plus />
                Add role
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Description</th>
                <th className="px-4 py-3 font-medium">Modules</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {roles.map((r) => (
                <tr key={r.id} className="border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3 font-medium">{r.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{r.description}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {r.moduleIds.map((id) => <Badge key={id} variant="secondary">{moduleName(id)}</Badge>)}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button variant="ghost" size="icon" onClick={() => remove(r.id)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </td>
                </tr>
              ))}
              {roles.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">No roles yet.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
