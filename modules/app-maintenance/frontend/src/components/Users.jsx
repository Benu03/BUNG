import { useEffect, useState } from 'react'
import { api } from '../api.js'

export default function Users() {
  const [users, setUsers] = useState([])
  const [roles, setRoles] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ username: '', fullName: '', email: '' })
  const [assigning, setAssigning] = useState(null) // user id currently editing roles for

  const load = () => Promise.all([api.listUsers(), api.listRoles()])
    .then(([u, r]) => { setUsers(u); setRoles(r) })
    .catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await api.createUser({ ...form, isActive: true, roleIds: [] })
      setForm({ username: '', fullName: '', email: '' })
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

  const roleName = (id) => roles.find((r) => r.id === id)?.name || id

  return (
    <div className="panel">
      {error && <div className="error">{error}</div>}

      <form className="inline-form" onSubmit={submit}>
        <label>
          Username
          <input type="text" required value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value })} />
        </label>
        <label>
          Full name
          <input type="text" required value={form.fullName}
            onChange={(e) => setForm({ ...form, fullName: e.target.value })} />
        </label>
        <label>
          Email
          <input type="email" required value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })} />
        </label>
        <button className="btn-primary" type="submit">Add User</button>
      </form>

      <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Username</th><th>Full name</th><th>Email</th><th>Roles</th><th></th></tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td>{u.username}</td>
              <td>{u.fullName}</td>
              <td className="muted">{u.email}</td>
              <td>
                {u.roleIds.map((id) => <span key={id} className="badge">{roleName(id)}</span>)}
                {assigning === u.id && (
                  <div style={{ marginTop: 6 }}>
                    {roles.map((r) => (
                      <label key={r.id} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginRight: 8, fontSize: 12 }}>
                        <input type="checkbox" checked={u.roleIds.includes(r.id)} onChange={() => toggleRole(u, r.id)} />
                        {r.name}
                      </label>
                    ))}
                  </div>
                )}
              </td>
              <td className="row-actions">
                <button className="btn-secondary" onClick={() => setAssigning(assigning === u.id ? null : u.id)}>
                  {assigning === u.id ? 'Done' : 'Assign roles'}
                </button>
                <button className="btn-danger" onClick={() => remove(u.id)}>Delete</button>
              </td>
            </tr>
          ))}
          {users.length === 0 && (
            <tr><td colSpan={5} className="muted">No users yet.</td></tr>
          )}
        </tbody>
      </table>
      </div>
    </div>
  )
}
