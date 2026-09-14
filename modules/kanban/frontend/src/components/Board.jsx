import { useEffect, useState } from 'react'
import { api } from '../api.js'

function Card({ card, columns, onChanged }) {
  const [title, setTitle] = useState(card.title)
  const [description, setDescription] = useState(card.description)

  const saveTitle = async () => {
    if (title === card.title) return
    await api.updateCard(card.id, { ...card, title })
    onChanged()
  }

  const saveDescription = async () => {
    if (description === card.description) return
    await api.updateCard(card.id, { ...card, description })
    onChanged()
  }

  const move = async (e) => {
    await api.moveCard(card.id, e.target.value, 0)
    onChanged()
  }

  const remove = async () => {
    if (!confirm('Delete this card?')) return
    await api.deleteCard(card.id)
    onChanged()
  }

  return (
    <div className="kanban-card">
      <input className="kanban-card-title" value={title}
        onChange={(e) => setTitle(e.target.value)} onBlur={saveTitle} />
      <textarea className="kanban-card-desc" rows={2} placeholder="Description"
        value={description} onChange={(e) => setDescription(e.target.value)} onBlur={saveDescription} />
      <div className="kanban-card-actions">
        <select value={card.columnId} onChange={move}>
          {columns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <button className="btn-danger" onClick={remove}>Delete</button>
      </div>
    </div>
  )
}

function ColumnLane({ column, columns, onChanged }) {
  const [name, setName] = useState(column.name)
  const [newCardTitle, setNewCardTitle] = useState('')

  const saveName = async () => {
    if (name === column.name) return
    await api.updateColumn(column.id, { ...column, name })
    onChanged()
  }

  const addCard = async (e) => {
    e.preventDefault()
    if (!newCardTitle.trim()) return
    await api.createCard(column.id, { title: newCardTitle, description: '', position: column.cards.length })
    setNewCardTitle('')
    onChanged()
  }

  const removeColumn = async () => {
    if (!confirm('Delete this column and its cards?')) return
    await api.deleteColumn(column.id)
    onChanged()
  }

  return (
    <div className="kanban-column">
      <div className="kanban-column-header">
        <input value={name} onChange={(e) => setName(e.target.value)} onBlur={saveName} />
        <button className="btn-secondary" onClick={removeColumn}>Delete</button>
      </div>

      <div className="kanban-column-cards">
        {column.cards.map((card) => (
          <Card key={card.id} card={card} columns={columns} onChanged={onChanged} />
        ))}
        {column.cards.length === 0 && <div className="muted">No cards</div>}
      </div>

      <form className="kanban-add-card" onSubmit={addCard}>
        <input type="text" placeholder="New card title" value={newCardTitle}
          onChange={(e) => setNewCardTitle(e.target.value)} />
        <button className="btn-primary" type="submit">Add</button>
      </form>
    </div>
  )
}

export default function Board({ boardId, onBack }) {
  const [board, setBoard] = useState(null)
  const [error, setError] = useState('')
  const [newColumnName, setNewColumnName] = useState('')

  const load = () => api.getBoard(boardId).then(setBoard).catch((e) => setError(e.message))

  useEffect(() => { load() }, [boardId])

  const addColumn = async (e) => {
    e.preventDefault()
    if (!newColumnName.trim()) return
    await api.createColumn(boardId, { name: newColumnName, position: board.columns.length })
    setNewColumnName('')
    load()
  }

  if (error) return <div className="panel"><div className="error">{error}</div></div>
  if (!board) return <div className="panel muted">Loading...</div>

  return (
    <div>
      <div className="toolbar">
        <button className="btn-secondary" onClick={onBack}>&larr; Boards</button>
        <h2 className="board-title">{board.name}</h2>
      </div>

      <div className="kanban-board">
        {board.columns.map((col) => (
          <ColumnLane key={col.id} column={col} columns={board.columns} onChanged={load} />
        ))}

        <form className="kanban-add-column" onSubmit={addColumn}>
          <input type="text" placeholder="New column name" value={newColumnName}
            onChange={(e) => setNewColumnName(e.target.value)} />
          <button className="btn-primary" type="submit">Add Column</button>
        </form>
      </div>
    </div>
  )
}
