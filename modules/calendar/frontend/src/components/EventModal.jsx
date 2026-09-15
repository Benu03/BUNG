import { useEffect, useState } from 'react'
import { Trash2, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'
import UserPicker from './UserPicker.jsx'

// datetime-local inputs work in the browser's local time, with no
// timezone info - these convert to/from that string and a real Date
// (which is what gets sent to the API as RFC3339/ISO, always in UTC).
function toLocalDateTimeValue(date) {
  const d = new Date(date)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
function toLocalDateValue(date) {
  return toLocalDateTimeValue(date).slice(0, 10)
}

function defaultStart() {
  const d = new Date()
  d.setMinutes(0, 0, 0)
  d.setHours(d.getHours() + 1)
  return d
}

export default function EventModal({ event, me, onClose, onSaved, onCreated, onDeleted }) {
  const { t } = useTranslation()
  const initialStart = event ? new Date(event.startAt) : defaultStart()
  const initialEnd = event ? new Date(event.endAt) : new Date(initialStart.getTime() + 60 * 60 * 1000)

  const [title, setTitle] = useState(event?.title || '')
  const [description, setDescription] = useState(event?.description || '')
  const [location, setLocation] = useState(event?.location || '')
  const [allDay, setAllDay] = useState(event?.allDay || false)
  const [start, setStart] = useState(allDay ? toLocalDateValue(initialStart) : toLocalDateTimeValue(initialStart))
  const [end, setEnd] = useState(allDay ? toLocalDateValue(initialEnd) : toLocalDateTimeValue(initialEnd))
  const [ownerId, setOwnerId] = useState(event?.ownerId)
  const [attendees, setAttendees] = useState([])
  const [allUsers, setAllUsers] = useState([])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  // currentId starts null when creating a brand-new event (no `event`
  // prop yet) and becomes set the moment the first Save succeeds - at
  // that point the modal switches into "edit" display (title, delete
  // button, attendees section) *without closing*, so you can invite
  // people to an event you just created without having to close and
  // reopen it. This was the actual bug behind "attendees doesn't work":
  // the whole attendees section used to be gated on the `event` prop
  // being present, which is never true while still creating.
  const [currentId, setCurrentId] = useState(event?.id || null)
  const isEdit = !!currentId

  // isOwner is only meaningful once `me` has loaded - defaults to true
  // while creating (the creator is always the owner) and false until we
  // know better while editing, so the read-only-for-attendees UI doesn't
  // briefly flash editable.
  const isOwner = !isEdit || (me != null && ownerId != null && me.id === ownerId)

  useEffect(() => {
    if (!event) return
    api.getEvent(event.id)
      .then((full) => {
        setOwnerId(full.ownerId)
        setAttendees(full.attendees || [])
      })
      .catch((e) => setError(e.message))
  }, [event?.id])

  // Fetched once regardless of create/edit mode, so the invite search box
  // is ready the instant an id exists (right after the first Save).
  useEffect(() => {
    api.listUsers().then(setAllUsers).catch(() => {})
  }, [])

  const toggleAllDay = (checked) => {
    setAllDay(checked)
    // Reformat the currently-entered values for the new input type instead
    // of losing them.
    setStart((v) => (checked ? v.slice(0, 10) : toLocalDateTimeValue(new Date(v))))
    setEnd((v) => (checked ? v.slice(0, 10) : toLocalDateTimeValue(new Date(v))))
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    if (!title.trim()) {
      setError(t('events.titleRequired'))
      return
    }
    const startAt = new Date(allDay ? `${start}T00:00:00` : start)
    const endAt = new Date(allDay ? `${end}T23:59:59` : end)
    if (endAt < startAt) {
      setError(t('events.endBeforeStart'))
      return
    }
    setSaving(true)
    try {
      const payload = {
        title: title.trim(),
        description,
        location,
        startAt: startAt.toISOString(),
        endAt: endAt.toISOString(),
        allDay,
      }
      if (isEdit) {
        const saved = await api.updateEvent(currentId, payload)
        onSaved(saved)
      } else {
        // First save of a brand-new event - stay open instead of closing,
        // so the attendees section (which needs a real event id) becomes
        // available right away.
        const saved = await api.createEvent(payload)
        setCurrentId(saved.id)
        setOwnerId(saved.ownerId)
        setAttendees([])
        onCreated(saved)
      }
    } catch (e) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!confirm(t('events.confirmDelete'))) return
    try {
      await api.deleteEvent(currentId)
      onDeleted(currentId)
    } catch (e) {
      setError(e.message)
    }
  }

  const invite = async (user) => {
    setError('')
    try {
      const attendee = await api.inviteAttendee(currentId, user.username)
      setAttendees((prev) => [...prev, attendee])
    } catch (e) {
      setError(e.message)
    }
  }

  const removeAttendee = async (userId) => {
    if (!confirm(t('events.confirmRemoveAttendee'))) return
    try {
      await api.removeAttendee(currentId, userId)
      setAttendees((prev) => prev.filter((a) => a.userId !== userId))
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-16" onClick={onClose}>
      <Card className="w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{isEdit ? t('events.editEvent') : t('events.newEvent')}</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose}><X className="h-4 w-4" /></Button>
        </CardHeader>
        <form onSubmit={submit}>
          <CardContent className="flex flex-col gap-4">
            {error && (
              <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
            )}

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ev-title">{t('events.title')}</Label>
              <Input id="ev-title" autoFocus value={title} onChange={(e) => setTitle(e.target.value)} disabled={!isOwner} />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ev-desc">{t('events.description')}</Label>
              <Input id="ev-desc" value={description} onChange={(e) => setDescription(e.target.value)} disabled={!isOwner} />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ev-loc">{t('events.location')}</Label>
              <Input id="ev-loc" value={location} onChange={(e) => setLocation(e.target.value)} disabled={!isOwner} />
            </div>

            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={allDay} onChange={(e) => toggleAllDay(e.target.checked)} disabled={!isOwner} className="h-4 w-4 rounded border-input" />
              {t('events.allDay')}
            </label>

            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="ev-start">{t('events.start')}</Label>
                <Input id="ev-start" type={allDay ? 'date' : 'datetime-local'} value={start} onChange={(e) => setStart(e.target.value)} disabled={!isOwner} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="ev-end">{t('events.end')}</Label>
                <Input id="ev-end" type={allDay ? 'date' : 'datetime-local'} value={end} onChange={(e) => setEnd(e.target.value)} disabled={!isOwner} />
              </div>
            </div>

            {isEdit && (
              <div className="flex flex-col gap-1.5 border-t pt-4">
                <Label>{t('events.attendees')}</Label>
                <div className="flex flex-col gap-1">
                  {attendees.map((a) => (
                    <div key={a.userId} className="flex items-center justify-between rounded-md border px-3 py-1.5 text-sm">
                      <span>{a.fullName || a.username} <span className="text-xs text-muted-foreground">@{a.username}</span></span>
                      {isOwner && (
                        <Button type="button" variant="ghost" size="icon" className="h-6 w-6" onClick={() => removeAttendee(a.userId)}>
                          <Trash2 className="h-3.5 w-3.5 text-destructive" />
                        </Button>
                      )}
                    </div>
                  ))}
                  {attendees.length === 0 && <p className="text-xs text-muted-foreground">-</p>}
                </div>
                {isOwner ? (
                  <div className="mt-1">
                    <UserPicker
                      users={allUsers}
                      excludeIds={[ownerId, ...attendees.map((a) => a.userId)]}
                      onPick={invite}
                      placeholder={t('events.inviteUsername')}
                    />
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">{t('events.ownerOnlyHint')}</p>
                )}
              </div>
            )}
          </CardContent>
          <div className="flex items-center justify-between border-t px-6 py-4">
            {isEdit && isOwner ? (
              <Button type="button" variant="ghost" onClick={remove} className="text-destructive hover:text-destructive">
                <Trash2 className="h-4 w-4" />
                {t('common.delete')}
              </Button>
            ) : <span />}
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={onClose}>{t('common.cancel')}</Button>
              {isOwner && <Button type="submit" disabled={saving}>{t('common.save')}</Button>}
            </div>
          </div>
        </form>
      </Card>
    </div>
  )
}
