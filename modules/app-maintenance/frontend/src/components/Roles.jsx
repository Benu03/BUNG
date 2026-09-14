import { useEffect, useState } from 'react'
import { api } from '../api.js'

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
    <div className="panel">
      {error && <div className="error">{error}</div>}

      <form className="inline-form" onSubmit={submit}>
        <label>
          Name
          <input type="text" required placeholder="Administrator"
            value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label>
          Description
          <input type="text" placeholder="optional"
            value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        <label>
          Modules
          <div>
            {modules.map((m) => (
              <label key={m.id} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginRight: 8, fontWeight: 'normal' }}>
                <input type="checkbox" checked={form.moduleIds.includes(m.id)} onChange={() => toggleModule(m.id)} />
                {m.name}
              </label>
            ))}
          </div>
        </label>
        <button className="btn-primary" type="submit" style={{ alignSelf: 'end' }}>Add Role</button>
      </form>

      <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Name</th><th>Description</th><th>Modules</th><th></th></tr>
        </thead>
        <tbody>
          {roles.map((r) => (
            <tr key={r.id}>
              <td>{r.name}</td>
              <td className="muted">{r.description}</td>
              <td>{r.moduleIds.map((id) => <span key={id} className="badge">{moduleName(id)}</span>)}</td>
              <td className="row-actions">
                <button className="btn-danger" onClick={() => remove(r.id)}>Delete</button>
              </td>
            </tr>
          ))}
          {roles.length === 0 && (
            <tr><td colSpan={4} className="muted">No roles yet.</td></tr>
          )}
        </tbody>
      </table>
      </div>
    </div>
  )
}
