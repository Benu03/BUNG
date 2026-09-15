import { useEffect, useRef, useState } from 'react'
import { Download, MessageCircle, Paperclip, Send } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import EmojiPicker from './EmojiPicker.jsx'
import { api } from '../api.js'
import { useTranslation } from '../lib/i18n.jsx'

function formatBytes(n) {
  if (n == null) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function formatTime(iso) {
  const d = new Date(iso)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { day: '2-digit', month: 'short' }) + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function Bubble({ msg, mine }) {
  const isImage = msg.attachmentFilename && msg.attachmentContentType?.startsWith('image/')
  return (
    <div className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[75%] rounded-2xl px-3 py-2 text-sm shadow-sm ${
          mine ? 'rounded-br-sm bg-primary text-primary-foreground' : 'rounded-bl-sm bg-card text-card-foreground'
        }`}
      >
        {msg.attachmentFilename && (
          isImage ? (
            <a href={api.downloadUrl(msg.id)} target="_blank" rel="noreferrer">
              <img src={api.downloadUrl(msg.id)} alt={msg.attachmentFilename} className="mb-1 max-h-56 rounded-lg object-cover" />
            </a>
          ) : (
            <a
              href={api.downloadUrl(msg.id)}
              className={`mb-1 flex items-center gap-2 rounded-lg border px-2.5 py-2 ${mine ? 'border-white/25 bg-white/10' : 'border-border bg-muted/50'}`}
            >
              <Paperclip className="h-4 w-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate text-xs font-medium">{msg.attachmentFilename}</span>
              <Download className="h-3.5 w-3.5 shrink-0" />
              <span className={`shrink-0 text-[10px] ${mine ? 'text-primary-foreground/70' : 'text-muted-foreground'}`}>{formatBytes(msg.attachmentSize)}</span>
            </a>
          )
        )}
        {msg.body && <p className="whitespace-pre-wrap break-words">{msg.body}</p>}
        <div className={`mt-0.5 text-right text-[10px] ${mine ? 'text-primary-foreground/70' : 'text-muted-foreground'}`}>
          {formatTime(msg.createdAt)}
        </div>
      </div>
    </div>
  )
}

export default function MessageThread({ me, conversation, messages, loading, onSend, onUpload }) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [uploading, setUploading] = useState(false)
  const bottomRef = useRef(null)
  const fileInputRef = useRef(null)

  useEffect(() => { bottomRef.current?.scrollIntoView({ block: 'end' }) }, [messages, conversation?.id])

  if (!conversation) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-muted-foreground">
        <MessageCircle className="h-12 w-12 opacity-30" />
        <p className="text-sm">{t('chat.selectConversation')}</p>
      </div>
    )
  }

  const submit = (e) => {
    e.preventDefault()
    const body = text.trim()
    if (!body) return
    onSend(body)
    setText('')
  }

  const onFilePicked = async (e) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setUploading(true)
    try {
      await onUpload(file)
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 items-center gap-2.5 border-b bg-card px-4 py-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-primary-dark text-sm font-semibold text-primary-foreground">
          {(conversation.friend.fullName || conversation.friend.username || '?').slice(0, 1).toUpperCase()}
        </div>
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">{conversation.friend.fullName || conversation.friend.username}</div>
          <div className="truncate text-xs text-muted-foreground">@{conversation.friend.username}</div>
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto px-4 py-4">
        {loading && <p className="text-center text-xs text-muted-foreground">{t('common.loading')}</p>}
        {!loading && messages.length === 0 && (
          <p className="text-center text-xs text-muted-foreground">{t('chat.noMessages')}</p>
        )}
        {messages.map((m) => (
          <Bubble key={m.id} msg={m} mine={m.senderId === me?.id} />
        ))}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={submit} className="flex shrink-0 items-center gap-1.5 border-t bg-card px-3 py-2.5">
        <input ref={fileInputRef} type="file" hidden onChange={onFilePicked} />
        <Button type="button" variant="ghost" size="icon" title={t('chat.attach')} disabled={uploading} onClick={() => fileInputRef.current?.click()}>
          <Paperclip className="h-4 w-4" />
        </Button>
        <EmojiPicker onPick={(e) => setText((v) => v + e)} />
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={t('chat.typeMessage')}
          className="flex-1"
          disabled={uploading}
        />
        <Button type="submit" size="icon" disabled={!text.trim() || uploading} title={t('chat.send')}>
          <Send className="h-4 w-4" />
        </Button>
      </form>
    </div>
  )
}
