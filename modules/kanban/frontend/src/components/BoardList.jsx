import { useEffect, useState } from 'react'
import { GripVertical, LayoutGrid, Plus, Trash2, TriangleAlert, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'

const DEFAULT_COLUMNS = ['To Do', 'In Progress', 'Done']

export default function BoardList({ onOpen }) {
  const [boards, setBoards] = useState([])
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [columns, setColumns] = useState(DEFAULT_COLUMNS)

  const load = () => api.listBoards().then(setBoards).catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const updateColumn = (i, value) => setColumns((cols) => cols.map((c, idx) => (idx === i ? value : c)))
  const addColumnField = () => setColumns((cols) => [...cols, ''])
  const removeColumnField = (i) => setColumns((cols) => cols.filter((_, idx) => idx !== i))

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      const board = await api.createBoard({ name, description, columns: columns.map((c) => c.trim()).filter(Boolean) })
      setName('')
      setDescription('')
      setColumns(DEFAULT_COLUMNS)
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
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Create a board</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="b-name">Board name</Label>
                <Input id="b-name" required value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="b-desc">Description</Label>
                <Input id="b-desc" placeholder="optional" value={description} onChange={(e) => setDescription(e.target.value)} />
              </div>
            </div>

            <div className="grid gap-1.5">
              <Label>Workflow (columns)</Label>
              <div className="flex flex-col gap-2">
                {columns.map((c, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <GripVertical className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <Input
                      value={c}
                      placeholder={`Stage ${i + 1}`}
                      onChange={(e) => updateColumn(i, e.target.value)}
                      className="h-8"
                    />
                    <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => removeColumnField(i)}>
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
              <Button type="button" variant="outline" size="sm" className="self-start" onClick={addColumnField}>
                <Plus />
                Add stage
              </Button>
              <p className="text-xs text-muted-foreground">Define the stages this board's cards move through - you can still add, rename or remove them later.</p>
            </div>

            <Button type="submit" className="self-start">
              <Plus />
              Create board
            </Button>
          </form>
        </CardContent>
      </Card>

      {boards.length === 0 ? (
        <Card className="flex flex-col items-center gap-2 border-dashed py-16 text-center">
          <LayoutGrid className="h-8 w-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No boards yet - create one above.</p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {boards.map((b) => (
            <Card key={b.id} onClick={() => onOpen(b.id)} className="cursor-pointer transition-colors hover:border-foreground/30">
              <CardHeader>
                <CardTitle className="text-base">{b.name}</CardTitle>
                {b.description && <p className="text-sm text-muted-foreground">{b.description}</p>}
              </CardHeader>
              <CardContent>
                <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={(e) => remove(b.id, e)}>
                  <Trash2 />
                  Delete
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
