import { useEffect, useState } from 'react'
import { api } from '../api.js'

export default function BoardList({ onOpen }) {
  const [boards, setBoards] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({ name: '', description: '' })

  const load = () => api.listBoards().then(setBoards).catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      const board = await api.createBoard(form)
      setForm({ name: '', description: '' })
      load()
      onOpen(board.id)
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (id, e) => {
    e.stopPropagation()
    if (!confirm('Delete this board and everything in it?')) return
    await api.deleteBoard(id)
    load()
  }

  return (
    <div className="panel">
      {error && <div className="error">{error}</div>}

      <form className="inline-form" onSubmit={submit}>
        <label>
          Board name
          <input type="text" required value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label>
          Description
          <input type="text" placeholder="optional" value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        <button className="btn-primary" type="submit">Create Board</button>
      </form>

      <div className="board-grid">
        {boards.map((b) => (
          <div key={b.id} className="board-card" onClick={() => onOpen(b.id)}>
            <div className="board-card-title">{b.name}</div>
            {b.description && <div className="muted">{b.description}</div>}
            <button className="btn-danger" onClick={(e) => remove(b.id, e)}>Delete</button>
          </div>
        ))}
        {boards.length === 0 && <div className="muted">No boards yet - create one above.</div>}
      </div>
    </div>
  )
}
