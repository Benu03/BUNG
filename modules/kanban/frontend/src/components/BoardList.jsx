import { useEffect, useState } from 'react'
import { LayoutGrid, Plus, Trash2, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'

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
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Create board</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
            <div className="grid gap-1.5">
              <Label htmlFor="b-name">Board name</Label>
              <Input id="b-name" required value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="b-desc">Description</Label>
              <Input id="b-desc" placeholder="optional" value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <Button type="submit">
              <Plus />
              Create
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
