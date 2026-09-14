import { useEffect, useState } from 'react'
import { api } from '../api.js'

export default function Modules() {
  const [modules, setModules] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ code: '', name: '', description: '' })

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
    if (!confirm('Delete this module?')) return
    await api.deleteModule(id)
    load()
  }

  return (
    <div className="panel">
      {error && <div className="error">{error}</div>}

      <form className="inline-form" onSubmit={submit}>
        <label>
          Code
          <input type="text" required placeholder="finance"
            value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
        </label>
        <label>
          Name
          <input type="text" required placeholder="Finance"
            value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label>
          Description
          <input type="text" placeholder="optional"
            value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        <button className="btn-primary" type="submit">Add Module</button>
      </form>

      <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Code</th><th>Name</th><th>Description</th><th>Active</th><th></th></tr>
        </thead>
        <tbody>
          {modules.map((m) => (
            <tr key={m.id}>
              <td>{m.code}</td>
              <td>{m.name}</td>
              <td className="muted">{m.description}</td>
              <td>{m.isActive ? 'Yes' : 'No'}</td>
              <td className="row-actions">
                <button className="btn-danger" onClick={() => remove(m.id)}>Delete</button>
              </td>
            </tr>
          ))}
          {modules.length === 0 && (
            <tr><td colSpan={5} className="muted">No modules yet.</td></tr>
          )}
        </tbody>
      </table>
      </div>
    </div>
  )
}
