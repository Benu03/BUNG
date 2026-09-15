import { useEffect, useRef, useState } from 'react'
import { File as FileIcon, Paperclip, Trash2, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'

const selectClass = 'h-9 rounded-md border border-input bg-background px-2 text-sm'

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function TicketModal({ ticketId, users, onClose, onSaved, onCreated, onDeleted }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(!!ticketId)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef(null)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState('')
  const [priority, setPriority] = useState('medium')
  const [status, setStatus] = useState('open')
  const [assigneeId, setAssigneeId] = useState('')
  const [requesterId, setRequesterId] = useState('')
  const [comments, setComments] = useState([])
  const [newComment, setNewComment] = useState('')
  const [attachments, setAttachments] = useState([])

  // currentId starts null for a brand-new ticket and is set the moment
  // the first Save succeeds - the modal then switches into edit display
  // in place (instead of closing) so comments/attachments can be added
  // right away, same fix as calendar's EventModal.
  const [currentId, setCurrentId] = useState(ticketId || null)
  const isEdit = !!currentId

  useEffect(() => {
    if (!ticketId) return
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
        setAttachments(tk.attachments || [])
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
      const payload = { title: title.trim(), description, category, priority }
      if (isEdit) {
        const saved = await api.updateTicket(currentId, { ...payload, status, assigneeId: assigneeId || null })
        onSaved(saved)
      } else {
        const saved = await api.createTicket(payload)
        setCurrentId(saved.id)
        setRequesterId(saved.requesterId)
        onCreated(saved)
      }
    } catch (e) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!confirm(t('tickets.confirmDelete'))) return
    try {
      await api.deleteTicket(currentId)
      onDeleted(currentId)
    } catch (e) {
      setError(e.message)
    }
  }

  const submitComment = async (e) => {
    e.preventDefault()
    if (!newComment.trim()) return
    try {
      const c = await api.addComment(currentId, newComment.trim())
      setComments((prev) => [...prev, c])
      setNewComment('')
    } catch (e) {
      setError(e.message)
    }
  }

  const doUpload = async (fileList) => {
    if (!fileList || fileList.length === 0) return
    setError('')
    setUploading(true)
    try {
      for (const file of fileList) {
        const a = await api.uploadAttachment(currentId, file)
        setAttachments((prev) => [...prev, a])
      }
    } catch (e) {
      setError(e.message)
    } finally {
      setUploading(false)
    }
  }

  const removeAttachment = async (attachmentId) => {
    if (!confirm(t('tickets.confirmDeleteAttachment'))) return
    try {
      await api.deleteAttachment(currentId, attachmentId)
      setAttachments((prev) => prev.filter((a) => a.id !== attachmentId))
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
                <div className="mb-2 flex items-center justify-between">
                  <h4 className="text-sm font-semibold">{t('tickets.attachments')}</h4>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading}
                  >
                    <Paperclip className="h-3.5 w-3.5" />
                    {uploading ? t('common.loading') : t('tickets.addAttachment')}
                  </Button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={(e) => doUpload(e.target.files)}
                  />
                </div>

                {attachments.length === 0 ? (
                  <p className="text-xs text-muted-foreground">{t('tickets.noAttachments')}</p>
                ) : (
                  <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
                    {attachments.map((a) => {
                      const isImage = a.contentType?.startsWith('image/')
                      const url = api.attachmentDownloadUrl(currentId, a.id)
                      return (
                        <div key={a.id} className="group relative overflow-hidden rounded-md border">
                          <a href={url} target="_blank" rel="noreferrer" className="block" title={a.filename}>
                            {isImage ? (
                              <img src={url} alt={a.filename} className="h-20 w-full object-cover" />
                            ) : (
                              <div className="flex h-20 w-full flex-col items-center justify-center gap-1 bg-muted/40 px-1 text-center">
                                <FileIcon className="h-5 w-5 text-muted-foreground" />
                                <span className="line-clamp-2 text-[10px] text-muted-foreground">{a.filename}</span>
                              </div>
                            )}
                          </a>
                          <div className="flex items-center justify-between gap-1 bg-card px-1.5 py-1">
                            <span className="truncate text-[10px] text-muted-foreground">{formatSize(a.size)}</span>
                            <button
                              type="button"
                              onClick={() => removeAttachment(a.id)}
                              className="rounded p-0.5 text-muted-foreground hover:text-destructive"
                            >
                              <Trash2 className="h-3 w-3" />
                            </button>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )}

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
