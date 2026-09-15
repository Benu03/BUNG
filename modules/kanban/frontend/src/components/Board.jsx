import { useEffect, useState } from 'react'
import { ArrowLeft, CalendarDays, Plus, Trash2, Users } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import MembersPanel from './Members.jsx'
import CardModal, { COLOR_STRIP } from './CardModal.jsx'
import { useTranslation } from '../lib/i18n.jsx'

function initials(name) {
  return name.split(' ').map((w) => w[0]).slice(0, 2).join('').toUpperCase()
}

// Overdue -> red, due within 2 days -> amber, otherwise a neutral chip -
// same "traffic light" convention as the due-date badges elsewhere in the
// app (e.g. ticketing's priority pills).
function formatDue(iso) {
  const due = new Date(iso)
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const diffDays = Math.round((due - today) / 86400000)
  const label = due.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  if (diffDays < 0) return { label, className: 'bg-rose-100 text-rose-700' }
  if (diffDays <= 2) return { label, className: 'bg-amber-100 text-amber-700' }
  return { label, className: 'bg-muted text-muted-foreground' }
}

function CardItem({ card, index, columnId, users, isDragging, onOpen, onDragStart, onDragOverCard, onDragEnd }) {
  const assignee = users.find((u) => u.id === card.assigneeId)
  const dueInfo = card.dueDate ? formatDue(card.dueDate) : null

  return (
    <div
      draggable
      onDragStart={(e) => { e.dataTransfer.setData('text/plain', card.id); onDragStart(card.id) }}
      onDragEnd={onDragEnd}
      onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); onDragOverCard(columnId, index) }}
      onClick={() => onOpen(card)}
      className={`group flex cursor-pointer flex-col overflow-hidden rounded-lg border bg-card shadow-sm transition-shadow hover:shadow-md ${isDragging ? 'opacity-40' : ''}`}
    >
      {card.color && <div className={`h-1.5 w-full ${COLOR_STRIP[card.color]}`} />}
      <div className="flex flex-col gap-2 p-2.5">
        <span className="text-sm font-medium">{card.title}</span>
        {(dueInfo || assignee) && (
          <div className="flex items-center justify-between gap-2">
            {dueInfo ? (
              <span className={`flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium ${dueInfo.className}`}>
                <CalendarDays className="h-3 w-3" />
                {dueInfo.label}
              </span>
            ) : <span />}
            {assignee && (
              <span
                title={assignee.fullName || assignee.username}
                className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-200 to-fuchsia-200 text-[9px] font-semibold text-slate-700"
              >
                {initials(assignee.fullName || assignee.username)}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function ColumnLane({ column, users, isDragOver, draggingCardId, onDragStart, onDragOverCard, onDragEnd, onDrop, onOpenCard, load }) {
  const { t } = useTranslation()
  const [name, setName] = useState(column.name)
  const [newCardTitle, setNewCardTitle] = useState('')

  const saveName = async () => {
    if (name === column.name) return
    await api.updateColumn(column.id, { ...column, name })
    load()
  }

  const addCard = async (e) => {
    e.preventDefault()
    if (!newCardTitle.trim()) return
    await api.createCard(column.id, { title: newCardTitle, description: '', position: column.cards.length })
    setNewCardTitle('')
    load()
  }

  const removeColumn = async () => {
    if (!confirm(t('board.confirmDeleteColumn'))) return
    await api.deleteColumn(column.id)
    load()
  }

  return (
    <div className={`flex w-72 shrink-0 flex-col gap-3 rounded-xl border bg-muted/40 p-3 transition-colors ${isDragOver ? 'border-primary bg-primary/5' : ''}`}>
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

      <div
        className="flex min-h-5 flex-col gap-2"
        onDragOver={(e) => { e.preventDefault(); if (column.cards.length === 0) onDragOverCard(column.id, 0) }}
        onDrop={(e) => { e.preventDefault(); onDrop(column.id) }}
      >
        {column.cards.map((card, i) => (
          <CardItem
            key={card.id}
            card={card}
            index={i}
            columnId={column.id}
            users={users}
            isDragging={draggingCardId === card.id}
            onOpen={onOpenCard}
            onDragStart={onDragStart}
            onDragOverCard={onDragOverCard}
            onDragEnd={onDragEnd}
          />
        ))}
        {column.cards.length === 0 && <div className="px-1 text-xs text-muted-foreground">{t('board.noCards')}</div>}
      </div>

      <form onSubmit={addCard} className="flex gap-1.5">
        <Input
          className="h-8 min-w-0 flex-1 rounded-md bg-background text-xs"
          placeholder={t('board.newCardTitle')}
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

export default function Board({ boardId, onBack, me }) {
  const { t } = useTranslation()
  const [board, setBoard] = useState(null)
  const [users, setUsers] = useState([])
  const [error, setError] = useState('')
  const [newColumnName, setNewColumnName] = useState('')
  const [showMembers, setShowMembers] = useState(false)
  const [live, setLive] = useState(false)
  const [selectedCard, setSelectedCard] = useState(null)
  const [draggingCardId, setDraggingCardId] = useState(null)
  const [dragOver, setDragOver] = useState(null) // { columnId, index }

  const load = () => api.getBoard(boardId).then(setBoard).catch((e) => setError(e.message))

  useEffect(() => { load() }, [boardId])
  useEffect(() => { api.listUsers().then(setUsers).catch(() => {}) }, [])

  // Real-time updates (see kanban-backend's hub.go): any other member's
  // change to this board arrives as a small event over this socket - we
  // don't bother reconciling a fine-grained patch, just refetch the whole
  // board, same as the local mutations above already do via onChanged.
  // Reconnects on its own (with a short delay) if the connection drops,
  // e.g. nginx/backend restart.
  useEffect(() => {
    if (!boardId) return
    let ws
    let reconnectTimer
    let closedByUs = false

    const connect = () => {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      ws = new WebSocket(`${proto}//${window.location.host}/kanban/api/boards/${boardId}/ws`)
      ws.onopen = () => setLive(true)
      ws.onmessage = () => load()
      ws.onclose = () => {
        setLive(false)
        if (!closedByUs) reconnectTimer = setTimeout(connect, 2000)
      }
      ws.onerror = () => ws.close()
    }
    connect()

    return () => {
      closedByUs = true
      clearTimeout(reconnectTimer)
      ws?.close()
    }
  }, [boardId])

  const addColumn = async (e) => {
    e.preventDefault()
    if (!newColumnName.trim()) return
    await api.createColumn(boardId, { name: newColumnName, position: board.columns.length })
    setNewColumnName('')
    load()
  }

  const deleteBoard = async () => {
    if (!confirm(t('board.confirmDeleteBoard'))) return
    await api.deleteBoard(boardId)
    onBack()
  }

  // ---- drag and drop ----
  // Cards are freely draggable within and across columns. Rather than
  // trying to compute one precise insertion position server-side, we
  // rebuild the destination column's whole card order client-side (using
  // the drop index we tracked while dragging) and re-send every card's
  // position in that order via moveCard - simple and correct, and cheap
  // enough for a board this size (same "just refetch/resend, don't be
  // clever" philosophy as the WebSocket broadcast payload).
  const handleDragStart = (cardId) => setDraggingCardId(cardId)
  const handleDragOverCard = (columnId, index) => setDragOver({ columnId, index })
  const handleDragEnd = () => { setDraggingCardId(null); setDragOver(null) }

  const handleDrop = async (columnId) => {
    const cardId = draggingCardId
    const target = dragOver
    handleDragEnd()
    if (!cardId || !target) return

    const destColumn = board.columns.find((c) => c.id === columnId)
    const draggedCard = board.columns.flatMap((c) => c.cards).find((c) => c.id === cardId)
    if (!destColumn || !draggedCard) return

    const destCards = destColumn.cards.filter((c) => c.id !== cardId)
    const insertAt = Math.min(target.index, destCards.length)
    destCards.splice(insertAt, 0, draggedCard)

    try {
      await Promise.all(destCards.map((c, i) => api.moveCard(c.id, columnId, i)))
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const handleCardChanged = () => {
    setSelectedCard(null)
    load()
  }
  const handleCardDeleted = () => {
    setSelectedCard(null)
    load()
  }

  if (error) return <Card className="p-4 text-sm text-destructive">{error}</Card>
  if (!board) return <div className="text-sm text-muted-foreground">{t('common.loading')}</div>

  const isOwner = me && board.ownerId === me.id

  return (
    <div>
      <div className="mb-5 flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={onBack}>
          <ArrowLeft />
          {t('board.boards')}
        </Button>
        <h2 className="flex flex-1 items-center gap-2 text-lg font-semibold tracking-tight">
          {board.name}
          <span
            title={live ? t('board.live') : t('board.reconnecting')}
            className={`h-1.5 w-1.5 rounded-full ${live ? 'bg-emerald-500' : 'bg-muted-foreground/40'}`}
          />
        </h2>
        <Button variant="outline" size="sm" onClick={() => setShowMembers(true)}>
          <Users />
          {t('board.members')}
        </Button>
        {isOwner && (
          <Button variant="outline" size="sm" className="text-destructive hover:text-destructive" onClick={deleteBoard}>
            <Trash2 />
            {t('board.deleteBoard')}
          </Button>
        )}
      </div>

      {showMembers && (
        <MembersPanel board={board} isOwner={isOwner} onClose={() => setShowMembers(false)} />
      )}

      {selectedCard && (
        <CardModal
          card={selectedCard}
          columns={board.columns}
          users={users}
          onClose={() => setSelectedCard(null)}
          onChanged={handleCardChanged}
          onDeleted={handleCardDeleted}
        />
      )}

      <div className="flex gap-4 overflow-x-auto pb-4">
        {board.columns.map((col) => (
          <ColumnLane
            key={col.id}
            column={col}
            users={users}
            isDragOver={dragOver?.columnId === col.id}
            draggingCardId={draggingCardId}
            onDragStart={handleDragStart}
            onDragOverCard={handleDragOverCard}
            onDragEnd={handleDragEnd}
            onDrop={handleDrop}
            onOpenCard={setSelectedCard}
            load={load}
          />
        ))}

        <form onSubmit={addColumn} className="flex w-64 shrink-0 gap-1.5 pt-1">
          <Input
            className="h-9"
            placeholder={t('board.newColumnName')}
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
