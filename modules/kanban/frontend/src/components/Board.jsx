import { useEffect, useState } from 'react'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'

function CardItem({ card, columns, onChanged }) {
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
    <div className="group flex flex-col gap-1.5 rounded-lg border bg-card p-2.5 shadow-sm transition-shadow hover:shadow-md">
      <input
        className="rounded border-0 bg-transparent px-0.5 text-sm font-medium outline-none focus:bg-accent"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onBlur={saveTitle}
      />
      <textarea
        className="resize-none rounded border-0 bg-transparent px-0.5 text-xs text-muted-foreground outline-none focus:bg-accent"
        rows={2}
        placeholder="Description"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        onBlur={saveDescription}
      />
      <div className="flex items-center justify-between gap-2 opacity-0 transition-opacity group-hover:opacity-100">
        <select
          value={card.columnId}
          onChange={move}
          className="h-7 flex-1 rounded-md border border-input bg-background px-1.5 text-xs"
        >
          {columns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={remove}>
          <Trash2 className="h-3.5 w-3.5 text-destructive" />
        </Button>
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
    <div className="flex w-72 shrink-0 flex-col gap-3 rounded-xl border bg-muted/40 p-3">
      <div className="flex items-center gap-1">
        <input
          className="min-w-0 flex-1 rounded border-0 bg-transparent px-1 py-0.5 text-sm font-semibold outline-none focus:bg-accent"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={saveName}
        />
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={removeColumn}>
          <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
        </Button>
      </div>

      <div className="flex min-h-5 flex-col gap-2">
        {column.cards.map((card) => (
          <CardItem key={card.id} card={card} columns={columns} onChanged={onChanged} />
        ))}
        {column.cards.length === 0 && <div className="px-1 text-xs text-muted-foreground">No cards</div>}
      </div>

      <form onSubmit={addCard} className="flex gap-1.5">
        <Input
          className="h-8 min-w-0 flex-1 rounded-md bg-background text-xs"
          placeholder="New card title"
          value={newCardTitle}
          onChange={(e) => setNewCardTitle(e.target.value)}
        />
        <Button type="submit" size="icon" className="h-8 w-8 shrink-0">
          <Plus className="h-3.5 w-3.5" />
        </Button>
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

  if (error) return <Card className="p-4 text-sm text-destructive">{error}</Card>
  if (!board) return <div className="text-sm text-muted-foreground">Loading...</div>

  return (
    <div>
      <div className="mb-5 flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={onBack}>
          <ArrowLeft />
          Boards
        </Button>
        <h2 className="text-lg font-semibold tracking-tight">{board.name}</h2>
      </div>

      <div className="flex gap-4 overflow-x-auto pb-4">
        {board.columns.map((col) => (
          <ColumnLane key={col.id} column={col} columns={board.columns} onChanged={load} />
        ))}

        <form onSubmit={addColumn} className="flex w-64 shrink-0 gap-1.5 pt-1">
          <Input
            className="h-9"
            placeholder="New column name"
            value={newColumnName}
            onChange={(e) => setNewColumnName(e.target.value)}
          />
          <Button type="submit" size="icon" className="shrink-0">
            <Plus />
          </Button>
        </form>
      </div>
    </div>
  )
}
