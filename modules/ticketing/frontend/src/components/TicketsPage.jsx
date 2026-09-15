import { useEffect, useMemo, useState } from 'react'
import { Search, Ticket as TicketIcon } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import { Badge } from './ui/badge.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'
import TicketModal from './TicketModal.jsx'

export const STATUS_STYLES = {
  open: 'bg-amber-100 text-amber-700',
  in_progress: 'bg-sky-100 text-sky-700',
  resolved: 'bg-emerald-100 text-emerald-700',
  closed: 'bg-slate-100 text-slate-600',
}
export const PRIORITY_STYLES = {
  low: 'bg-slate-100 text-slate-600',
  medium: 'bg-sky-100 text-sky-700',
  high: 'bg-amber-100 text-amber-700',
  urgent: 'bg-rose-100 text-rose-700',
}
export const STATUS_LABEL_KEYS = {
  open: 'tickets.statusOpen',
  in_progress: 'tickets.statusInProgress',
  resolved: 'tickets.statusResolved',
  closed: 'tickets.statusClosed',
}
export const PRIORITY_LABEL_KEYS = {
  low: 'tickets.priorityLow',
  medium: 'tickets.priorityMedium',
  high: 'tickets.priorityHigh',
  urgent: 'tickets.priorityUrgent',
}

export default function TicketsPage() {
  const { t } = useTranslation()
  const [tickets, setTickets] = useState([])
  const [users, setUsers] = useState([])
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [modalTicket, setModalTicket] = useState(undefined) // undefined = closed, null = create, id string = edit

  const load = () => {
    Promise.all([api.listTickets(), api.listUsers()])
      .then(([ts, us]) => { setTickets(ts); setUsers(us) })
      .catch((e) => setError(e.message))
  }

  useEffect(() => { load() }, [])

  const userName = (id) => {
    const u = users.find((u) => u.id === id)
    return u ? (u.fullName || u.username) : t('common.unassigned')
  }

  const q = filter.trim().toLowerCase()
  const filtered = useMemo(
    () => tickets
      .filter((tk) => !statusFilter || tk.status === statusFilter)
      .filter((tk) => !q || [tk.title, tk.description, tk.category].some((v) => v?.toLowerCase().includes(q))),
    [tickets, q, statusFilter]
  )

  const onSaved = (saved) => {
    setTickets((prev) => {
      const exists = prev.some((tk) => tk.id === saved.id)
      return exists ? prev.map((tk) => (tk.id === saved.id ? saved : tk)) : [saved, ...prev]
    })
    setModalTicket(undefined)
  }
  const onDeleted = (id) => {
    setTickets((prev) => prev.filter((tk) => tk.id !== id))
    setModalTicket(undefined)
  }

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2">
            <Search className="h-4 w-4 text-muted-foreground" />
            <Input
              placeholder={t('common.filterPlaceholder')}
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="h-8 max-w-xs border-0 shadow-none focus-visible:ring-0"
            />
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-2 text-sm"
          >
            <option value="">{t('tickets.allStatuses')}</option>
            <option value="open">{t('tickets.statusOpen')}</option>
            <option value="in_progress">{t('tickets.statusInProgress')}</option>
            <option value="resolved">{t('tickets.statusResolved')}</option>
            <option value="closed">{t('tickets.statusClosed')}</option>
          </select>
        </div>
        <Button size="sm" onClick={() => setModalTicket(null)}>
          <TicketIcon className="h-4 w-4" />
          {t('tickets.newTicket')}
        </Button>
      </div>

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">{t('tickets.title')}</th>
                <th className="px-4 py-3 font-medium">{t('tickets.status')}</th>
                <th className="px-4 py-3 font-medium">{t('tickets.priority')}</th>
                <th className="px-4 py-3 font-medium">{t('tickets.assignee')}</th>
                <th className="px-4 py-3 font-medium">{t('tickets.requester')}</th>
                <th className="px-4 py-3 font-medium">{t('tickets.created')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((tk) => (
                <tr key={tk.id} onClick={() => setModalTicket(tk.id)} className="cursor-pointer border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3">
                    <div className="font-medium">{tk.title}</div>
                    {tk.category && <div className="text-xs text-muted-foreground">{tk.category}</div>}
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant="outline" className={`border-transparent font-medium ${STATUS_STYLES[tk.status]}`}>
                      {t(STATUS_LABEL_KEYS[tk.status])}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant="outline" className={`border-transparent font-medium ${PRIORITY_STYLES[tk.priority]}`}>
                      {t(PRIORITY_LABEL_KEYS[tk.priority])}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {tk.assigneeId ? userName(tk.assigneeId) : <span className="italic">{t('common.unassigned')}</span>}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{userName(tk.requesterId)}</td>
                  <td className="px-4 py-3 text-muted-foreground">{new Date(tk.createdAt).toLocaleDateString()}</td>
                </tr>
              ))}
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-10 text-center text-muted-foreground">
                    {q || statusFilter ? t('common.noMatches') : t('tickets.empty')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <p className="text-center text-xs text-muted-foreground">{t('tickets.footerHint')}</p>

      {modalTicket !== undefined && (
        <TicketModal
          ticketId={modalTicket}
          users={users}
          onClose={() => setModalTicket(undefined)}
          onSaved={onSaved}
          onDeleted={onDeleted}
        />
      )}
    </div>
  )
}
