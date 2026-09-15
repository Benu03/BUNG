import { useEffect, useRef, useState } from 'react'
import { Bell, Check } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'
import { useTranslation } from '../lib/i18n.jsx'

// Cross-module by design: fetches straight from /notifications/api/ (a
// different module's gateway path) rather than each app's own api.js -
// this same file is duplicated verbatim into every frontend's header
// (portal, app-maintenance, kanban, my-storage, notifications itself),
// same pattern as ThemeToggle/LanguageToggle. The notifications gateway
// locations don't require any specific module grant (see
// /modules/notifications/nginx.conf), so this works for any logged-in
// user regardless of which modules they have access to.
const API_BASE = '/notifications/api/'
const POLL_MS = 30000

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, { headers: { 'Content-Type': 'application/json' }, ...options })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  if (res.status === 204) return null
  return res.json()
}

function timeAgo(iso) {
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (seconds < 60) return 'now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

export default function NotificationBell({ className }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState([])
  const [unread, setUnread] = useState(0)
  const ref = useRef(null)

  const loadCount = () => request('notifications/unread-count').then((d) => setUnread(d.count)).catch(() => {})
  const loadItems = () => request('notifications?limit=8').then(setItems).catch(() => {})

  useEffect(() => {
    loadCount()
    const id = setInterval(loadCount, POLL_MS)
    return () => clearInterval(id)
  }, [])

  useEffect(() => {
    if (open) loadItems()
  }, [open])

  useEffect(() => {
    const onClickOutside = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false) }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [])

  const openItem = async (n) => {
    if (!n.readAt) {
      await request(`notifications/${n.id}/read`, { method: 'POST' }).catch(() => {})
      setItems((prev) => prev.map((it) => (it.id === n.id ? { ...it, readAt: new Date().toISOString() } : it)))
      setUnread((c) => Math.max(0, c - 1))
    }
    if (n.link) window.location.href = n.link
  }

  const markAllRead = async (e) => {
    e.stopPropagation()
    await request('notifications/read-all', { method: 'POST' }).catch(() => {})
    setItems((prev) => prev.map((it) => ({ ...it, readAt: it.readAt || new Date().toISOString() })))
    setUnread(0)
  }

  return (
    <div className="relative" ref={ref}>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setOpen((v) => !v)}
        title={t('bell.title')}
        className={className}
      >
        <span className="relative">
          <Bell className="h-4 w-4" />
          {unread > 0 && (
            <span className="absolute -right-1.5 -top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold text-destructive-foreground">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </span>
      </Button>

      {open && (
        <Card className="absolute right-0 top-full z-30 mt-2 w-80 overflow-hidden text-foreground shadow-xl">
          <div className="flex items-center justify-between border-b px-3 py-2">
            <span className="text-sm font-semibold">{t('bell.title')}</span>
            {unread > 0 && (
              <button onClick={markAllRead} className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                <Check className="h-3 w-3" />
                {t('bell.markAllRead')}
              </button>
            )}
          </div>
          <div className="max-h-80 overflow-y-auto">
            {items.length === 0 && (
              <p className="px-3 py-6 text-center text-xs text-muted-foreground">{t('bell.empty')}</p>
            )}
            {items.map((n) => (
              <button
                key={n.id}
                onClick={() => openItem(n)}
                className="flex w-full flex-col gap-0.5 border-b px-3 py-2.5 text-left last:border-0 hover:bg-muted/50"
              >
                <div className="flex items-start justify-between gap-2">
                  <span className={`text-xs ${n.readAt ? 'font-normal text-muted-foreground' : 'font-semibold'}`}>{n.title}</span>
                  <span className="shrink-0 text-[10px] text-muted-foreground">{timeAgo(n.createdAt)}</span>
                </div>
                {n.body && <span className="text-xs text-muted-foreground">{n.body}</span>}
                {!n.readAt && <span className="mt-0.5 h-1.5 w-1.5 rounded-full bg-primary" />}
              </button>
            ))}
          </div>
          <a href="/notifications/" className="block border-t px-3 py-2 text-center text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground">
            {t('bell.viewAll')}
          </a>
        </Card>
      )}
    </div>
  )
}
