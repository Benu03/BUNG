import { useState } from 'react'
import { Trash2, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Label } from './ui/label.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import UserPicker from './UserPicker.jsx'
import { useTranslation } from '../lib/i18n.jsx'

export const CARD_COLORS = ['', 'rose', 'amber', 'emerald', 'sky', 'violet']
// Swatch (picker) and badge (compact card strip) share the same hues, just
// different shades - swatches a bit bolder so they're pickable at a
// glance, the strip on the card itself stays subtle/pastel like the rest
// of the UI.
export const COLOR_SWATCH = {
  rose: 'bg-rose-300', amber: 'bg-amber-300', emerald: 'bg-emerald-300', sky: 'bg-sky-300', violet: 'bg-violet-300',
}
export const COLOR_STRIP = {
  rose: 'bg-rose-200', amber: 'bg-amber-200', emerald: 'bg-emerald-200', sky: 'bg-sky-200', violet: 'bg-violet-200',
}

function toLocalDateValue(date) {
  const d = new Date(date)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export default function CardModal({ card, columns, users, onClose, onChanged, onDeleted }) {
  const { t } = useTranslation()
  const [title, setTitle] = useState(card.title)
  const [description, setDescription] = useState(card.description)
  const [columnId, setColumnId] = useState(card.columnId)
  const [assigneeId, setAssigneeId] = useState(card.assigneeId || '')
  const [dueDate, setDueDate] = useState(card.dueDate ? toLocalDateValue(card.dueDate) : '')
  const [color, setColor] = useState(card.color || '')
  const [saving, setSaving] = useState(false)

  const assignee = users.find((u) => u.id === assigneeId)

  const save = async (patch) => {
    setSaving(true)
    try {
      const payload = {
        title, description, position: card.position,
        assigneeId: assigneeId || null,
        dueDate: dueDate ? new Date(`${dueDate}T00:00:00`).toISOString() : null,
        color,
        ...patch,
      }
      const updated = await api.updateCard(card.id, payload)
      if (columnId !== card.columnId) {
        await api.moveCard(card.id, columnId, 0)
      }
      onChanged(updated)
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!confirm(t('board.confirmDeleteCard'))) return
    await api.deleteCard(card.id)
    onDeleted(card.id)
  }

  return (
    <div className="fixed inset-0 z-30 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-16" onClick={onClose}>
      <Card className="w-full max-w-lg" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{t('board.cardDetails')}</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose}><X className="h-4 w-4" /></Button>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="card-title">{t('board.cardTitle')}</Label>
            <Input id="card-title" value={title} onChange={(e) => setTitle(e.target.value)} onBlur={() => save({ title })} />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="card-desc">{t('common.description')}</Label>
            <textarea
              id="card-desc"
              rows={3}
              className="resize-none rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              onBlur={() => save({ description })}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="card-column">{t('board.column')}</Label>
              <select
                id="card-column"
                value={columnId}
                onChange={(e) => { setColumnId(e.target.value); save({ position: 0 }) }}
                className="h-9 rounded-md border border-input bg-background px-2 text-sm"
              >
                {columns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="card-due">{t('board.dueDate')}</Label>
              <Input
                id="card-due"
                type="date"
                value={dueDate}
                onChange={(e) => { setDueDate(e.target.value); save({}) }}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t('board.color')}</Label>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => { setColor(''); save({ color: '' }) }}
                className={`h-6 w-6 rounded-full border ${color === '' ? 'ring-2 ring-primary ring-offset-1' : ''}`}
                title={t('board.colorNone')}
              />
              {CARD_COLORS.filter(Boolean).map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => { setColor(c); save({ color: c }) }}
                  className={`h-6 w-6 rounded-full ${COLOR_SWATCH[c]} ${color === c ? 'ring-2 ring-primary ring-offset-1' : ''}`}
                  title={c}
                />
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t('board.assignee')}</Label>
            {assignee ? (
              <div className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                <span>{assignee.fullName || assignee.username} <span className="text-xs text-muted-foreground">@{assignee.username}</span></span>
                <Button
                  type="button" variant="ghost" size="icon" className="h-6 w-6"
                  onClick={() => { setAssigneeId(''); save({ assigneeId: null }) }}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </div>
            ) : (
              <UserPicker
                users={users}
                onPick={(u) => { setAssigneeId(u.id); save({ assigneeId: u.id }) }}
                placeholder={t('board.assignSomeone')}
              />
            )}
          </div>
        </CardContent>
        <div className="flex items-center justify-between border-t px-6 py-4">
          <Button type="button" variant="ghost" onClick={remove} className="text-destructive hover:text-destructive">
            <Trash2 className="h-4 w-4" />
            {t('common.delete')}
          </Button>
          <Button type="button" variant="outline" onClick={onClose} disabled={saving}>{t('common.close')}</Button>
        </div>
      </Card>
    </div>
  )
}
