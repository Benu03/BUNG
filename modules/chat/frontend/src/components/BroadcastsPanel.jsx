import { useState } from 'react'
import { Megaphone, Send } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { useTranslation } from '../lib/i18n.jsx'

function formatTime(iso) {
  return new Date(iso).toLocaleString([], { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}

// Admin-only broadcasts to the whole team - listed newest first (see
// backend store.go). The composer only renders for app-maintenance admins;
// the backend enforces the same restriction independently in createBroadcast,
// this is purely so non-admins don't see a control that would 403.
export default function BroadcastsPanel({ isAdmin, broadcasts, onSend }) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    const body = text.trim()
    if (!body) return
    setSending(true)
    try {
      await onSend(body)
      setText('')
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b bg-card px-4 py-3">
        <Megaphone className="h-4 w-4 text-primary" />
        <span className="text-sm font-semibold">{t('chat.broadcasts')}</span>
      </div>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-4">
        {broadcasts.length === 0 && (
          <p className="text-center text-xs text-muted-foreground">{t('chat.broadcastEmpty')}</p>
        )}
        {broadcasts.map((b) => (
          <div key={b.id} className="rounded-xl border bg-card px-3.5 py-3 shadow-sm">
            <div className="mb-1 flex items-center justify-between gap-2">
              <span className="flex items-center gap-1.5 text-xs font-semibold text-primary">
                <Megaphone className="h-3 w-3" />
                {b.sender.fullName || b.sender.username}
              </span>
              <span className="shrink-0 text-[10px] text-muted-foreground">{formatTime(b.createdAt)}</span>
            </div>
            <p className="whitespace-pre-wrap break-words text-sm">{b.body}</p>
          </div>
        ))}
      </div>

      {isAdmin && (
        <form onSubmit={submit} className="flex shrink-0 items-center gap-1.5 border-t bg-card px-3 py-2.5">
          <Input
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={t('chat.broadcastComposerPlaceholder')}
            className="flex-1"
            disabled={sending}
          />
          <Button type="submit" size="sm" disabled={!text.trim() || sending}>
            <Send className="h-4 w-4" />
            {t('chat.broadcastSend')}
          </Button>
        </form>
      )}
    </div>
  )
}
