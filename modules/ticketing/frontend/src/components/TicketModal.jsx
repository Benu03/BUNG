import { useEffect, useState } from 'react'
import { Trash2, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'

const selectClass = 'h-9 rounded-md border border-input bg-background px-2 text-sm'

export default function TicketModal({ ticketId, users, onClose, onSaved, onDeleted }) {
  const { t } = useTranslation()
  const isEdit = !!ticketId
  const [loading, setLoading] = useState(isEdit)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState('')
  const [priority, setPriority] = useState('medium')
  const [status, setStatus] = useState('open')
  const [assigneeId, setAssigneeId] = useState('')
  const [requesterId, setRequesterId] = useState('')
  const [comments, setComments] = useState([])
  const [newComment, setNewComment] = useState('')

  useEffect(() => {
    if (!isEdit) return
    api.getTicket(ticketId)
      .then((tk) => {
        setTitle(tk.title)
        setDescription(tk.description)
        setCategory(tk.category)
        setPriority(tk.priority)
        setStatus(tk.status)
        setAssigneeId(tk.assigneeId || '')
        setRequesterId(tk.requesterId)
        setComments(tk.comments || [])
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [ticketId])

  const userName = (id) => {
    const u = users.find((u) => u.id === id)
    return u ? (u.fullName || u.username) : id
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    if (!title.trim()) {
      setError(t('tickets.titleRequired'))
      return
    }
    setSaving(true)
    try {
      const saved = isEdit
        ? await api.updateTicket(ticketId, { title: title.trim(), description, category, priority, status, assigneeId: assigneeId || null })
        : await api.createTicket({ title: title.trim(), description, category, priority })
      onSaved(saved)
    } catch (e) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!confirm(t('tickets.confirmDelete'))) return
    try {
      await api.deleteTicket(ticketId)
      onDeleted(ticketId)
    } catch (e) {
      setError(e.message)
    }
  }

  const submitComment = async (e) => {
    e.preventDefault()
    if (!newComment.trim()) return
    try {
      const c = await api.addComment(ticketId, newComment.trim())
      setComments((prev) => [...prev, c])
      setNewComment('')
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-12" onClick={onClose}>
      <Card className="w-full max-w-lg" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{isEdit ? title || '...' : t('tickets.newTicket')}</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose}><X className="h-4 w-4" /></Button>
        </CardHeader>

        {loading ? (
          <CardContent><p className="text-sm text-muted-foreground">{t('common.loading')}</p></CardContent>
        ) : (
          <>
            <form onSubmit={submit}>
              <CardContent className="flex flex-col gap-4">
                {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="tk-title">{t('tickets.title')}</Label>
                  <Input id="tk-title" autoFocus value={title} onChange={(e) => setTitle(e.target.value)} />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="tk-desc">{t('tickets.description')}</Label>
                  <Input id="tk-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="tk-cat">{t('tickets.category')}</Label>
                    <Input id="tk-cat" value={category} onChange={(e) => setCategory(e.target.value)} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="tk-prio">{t('tickets.priority')}</Label>
                    <select id="tk-prio" value={priority} onChange={(e) => setPriority(e.target.value)} className={selectClass}>
                      <option value="low">{t('tickets.priorityLow')}</option>
                      <option value="medium">{t('tickets.priorityMedium')}</option>
                      <option value="high">{t('tickets.priorityHigh')}</option>
                      <option value="urgent">{t('tickets.priorityUrgent')}</option>
                    </select>
                  </div>
                </div>

                {isEdit && (
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor="tk-status">{t('tickets.status')}</Label>
                      <select id="tk-status" value={status} onChange={(e) => setStatus(e.target.value)} className={selectClass}>
                        <option value="open">{t('tickets.statusOpen')}</option>
                        <option value="in_progress">{t('tickets.statusInProgress')}</option>
                        <option value="resolved">{t('tickets.statusResolved')}</option>
                        <option value="closed">{t('tickets.statusClosed')}</option>
                      </select>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor="tk-assignee">{t('tickets.assignee')}</Label>
                      <select id="tk-assignee" value={assigneeId} onChange={(e) => setAssigneeId(e.target.value)} className={selectClass}>
                        <option value="">{t('common.unassigned')}</option>
                        {users.map((u) => <option key={u.id} value={u.id}>{u.fullName || u.username}</option>)}
                      </select>
                    </div>
                  </div>
                )}

                {isEdit && (
                  <p className="text-xs text-muted-foreground">{t('tickets.requester')}: {userName(requesterId)}</p>
                )}
              </CardContent>
              <div className="flex items-center justify-between border-t px-6 py-4">
                {isEdit ? (
                  <Button type="button" variant="ghost" onClick={remove} className="text-destructive hover:text-destructive">
                    <Trash2 className="h-4 w-4" />
                    {t('common.delete')}
                  </Button>
                ) : <span />}
                <div className="flex gap-2">
                  <Button type="button" variant="outline" onClick={onClose}>{t('common.cancel')}</Button>
                  <Button type="submit" disabled={saving}>{t('common.save')}</Button>
                </div>
              </div>
            </form>

            {isEdit && (
              <div className="border-t px-6 py-4">
                <h4 className="mb-2 text-sm font-semibold">{t('tickets.comments')}</h4>
                <div className="mb-3 flex max-h-48 flex-col gap-2 overflow-y-auto">
                  {comments.length === 0 && <p className="text-xs text-muted-foreground">{t('tickets.noComments')}</p>}
                  {comments.map((c) => (
                    <div key={c.id} className="rounded-md border bg-muted/30 px-3 py-2 text-sm">
                      <div className="mb-0.5 flex items-center justify-between text-xs text-muted-foreground">
                        <span className="font-medium text-foreground">{userName(c.authorId)}</span>
                        <span>{new Date(c.createdAt).toLocaleString()}</span>
                      </div>
                      {c.body}
                    </div>
                  ))}
                </div>
                <form onSubmit={submitComment} className="flex gap-2">
                  <Input
                    placeholder={t('tickets.commentPlaceholder')}
                    value={newComment}
                    onChange={(e) => setNewComment(e.target.value)}
                  />
                  <Button type="submit" size="sm" className="shrink-0">{t('tickets.addComment')}</Button>
                </form>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  )
}
