import { useEffect, useMemo, useState } from 'react'
import { CalendarPlus, Clock, MapPin, Search } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'
import EventModal from './EventModal.jsx'

function dateKey(iso) {
  return new Date(iso).toDateString()
}

function formatDateHeader(iso, lang) {
  return new Date(iso).toLocaleDateString(lang === 'id' ? 'id-ID' : 'en-US', {
    weekday: 'long', year: 'numeric', month: 'long', day: 'numeric',
  })
}

function formatTimeRange(ev) {
  if (ev.allDay) return null
  const opts = { hour: '2-digit', minute: '2-digit' }
  return `${new Date(ev.startAt).toLocaleTimeString([], opts)} - ${new Date(ev.endAt).toLocaleTimeString([], opts)}`
}

export default function EventsPage({ me }) {
  const { t, lang } = useTranslation()
  const [events, setEvents] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')
  const [modalEvent, setModalEvent] = useState(undefined) // undefined = closed, null = create, object = edit

  const load = () => {
    setLoading(true)
    api.listEvents().then(setEvents).catch((e) => setError(e.message)).finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  // Deep link from the global command palette (?event=<id>) - resolve the
  // full event (EventModal needs more than just an id, unlike ticket/board
  // deep links) then open it, and strip the param so a refresh doesn't
  // reopen it.
  useEffect(() => {
    const id = new URLSearchParams(window.location.search).get('event')
    if (!id) return
    api.getEvent(id).then(setModalEvent).catch(() => {})
    const url = new URL(window.location.href)
    url.searchParams.delete('event')
    window.history.replaceState({}, '', url)
  }, [])

  const q = filter.trim().toLowerCase()
  const filtered = useMemo(
    () => events.filter((e) => !q || [e.title, e.description, e.location].some((v) => v?.toLowerCase().includes(q))),
    [events, q]
  )

  const groups = useMemo(() => {
    const map = new Map()
    for (const e of filtered) {
      const key = dateKey(e.startAt)
      if (!map.has(key)) map.set(key, [])
      map.get(key).push(e)
    }
    return [...map.entries()]
  }, [filtered])

  const onSaved = (saved) => {
    setEvents((prev) => {
      const exists = prev.some((e) => e.id === saved.id)
      return exists ? prev.map((e) => (e.id === saved.id ? saved : e)) : [...prev, saved]
    })
    setModalEvent(undefined)
  }

  // Called after the *first* save of a brand-new event - unlike onSaved,
  // this deliberately leaves the modal open (see EventModal) so the user
  // can invite attendees right away, without an extra close-and-reopen.
  const onCreated = (saved) => {
    setEvents((prev) => [...prev, saved])
  }

  const onDeleted = (id) => {
    setEvents((prev) => prev.filter((e) => e.id !== id))
    setModalEvent(undefined)
  }

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('common.filterPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="h-8 max-w-xs border-0 shadow-none focus-visible:ring-0"
          />
        </div>
        <Button size="sm" onClick={() => setModalEvent(null)}>
          <CalendarPlus className="h-4 w-4" />
          {t('events.newEvent')}
        </Button>
      </div>

      {!loading && groups.length === 0 && (
        <Card className="flex flex-col items-center gap-2 rounded-2xl border-dashed py-16 text-center text-muted-foreground">
          {q ? t('common.noMatches') : t('events.empty')}
        </Card>
      )}

      <div className="flex flex-col gap-4">
        {groups.map(([key, dayEvents]) => (
          <div key={key}>
            <h3 className="mb-2 text-sm font-semibold text-muted-foreground">
              {formatDateHeader(dayEvents[0].startAt, lang)}
            </h3>
            <Card className="divide-y overflow-hidden">
              {dayEvents.map((ev) => (
                <button
                  key={ev.id}
                  onClick={() => setModalEvent(ev)}
                  className="flex w-full flex-col gap-1 px-4 py-3 text-left hover:bg-muted/40"
                >
                  <span className="flex items-center gap-2 font-medium">
                    {ev.title}
                    {me && ev.ownerId !== me.id && (
                      <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-normal text-muted-foreground">
                        {t('events.invited')}
                      </span>
                    )}
                  </span>
                  <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {ev.allDay ? t('events.allDay') : formatTimeRange(ev)}
                    </span>
                    {ev.location && (
                      <span className="flex items-center gap-1">
                        <MapPin className="h-3 w-3" />
                        {ev.location}
                      </span>
                    )}
                  </div>
                </button>
              ))}
            </Card>
          </div>
        ))}
      </div>

      <p className="text-center text-xs text-muted-foreground">{t('events.footerHint')}</p>

      {modalEvent !== undefined && (
        <EventModal
          event={modalEvent}
          me={me}
          onClose={() => setModalEvent(undefined)}
          onSaved={onSaved}
          onCreated={onCreated}
          onDeleted={onDeleted}
        />
      )}
    </div>
  )
}
