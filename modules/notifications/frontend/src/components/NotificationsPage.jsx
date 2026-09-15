import { useEffect, useState } from 'react'
import { Bell, Check } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'
import { useTranslation } from '../lib/i18n.jsx'

const API_BASE = '/notifications/api/'
const PAGE_SIZE = 30

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, { headers: { 'Content-Type': 'application/json' }, ...options })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  if (res.status === 204) return null
  return res.json()
}

export default function NotificationsPage() {
  const { t } = useTranslation()
  const [items, setItems] = useState([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)

  const loadPage = async (nextOffset) => {
    setLoading(true)
    try {
      const page = await request(`notifications?limit=${PAGE_SIZE}&offset=${nextOffset}`)
      setItems((prev) => (nextOffset === 0 ? page : [...prev, ...page]))
      setHasMore(page.length === PAGE_SIZE)
      setOffset(nextOffset + page.length)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadPage(0) }, [])

  const openItem = async (n) => {
    if (!n.readAt) {
      await request(`notifications/${n.id}/read`, { method: 'POST' }).catch(() => {})
      setItems((prev) => prev.map((it) => (it.id === n.id ? { ...it, readAt: new Date().toISOString() } : it)))
    }
    if (n.link) window.location.href = n.link
  }

  const markAllRead = async () => {
    await request('notifications/read-all', { method: 'POST' }).catch(() => {})
    setItems((prev) => prev.map((it) => ({ ...it, readAt: it.readAt || new Date().toISOString() })))
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button variant="outline" size="sm" onClick={markAllRead}>
          <Check />
          {t('page.markAllRead')}
        </Button>
      </div>

      <Card className="divide-y overflow-hidden">
        {items.length === 0 && !loading && (
          <div className="flex flex-col items-center gap-2 py-16 text-center text-muted-foreground">
            <Bell className="h-8 w-8" />
            <p className="text-sm">{t('page.empty')}</p>
          </div>
        )}
        {items.map((n) => (
          <button
            key={n.id}
            onClick={() => openItem(n)}
            className="flex w-full items-start justify-between gap-3 px-4 py-3 text-left hover:bg-muted/40"
          >
            <div>
              <div className={`text-sm ${n.readAt ? 'font-normal text-muted-foreground' : 'font-semibold'}`}>{n.title}</div>
              {n.body && <div className="text-sm text-muted-foreground">{n.body}</div>}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {!n.readAt && <span className="h-2 w-2 rounded-full bg-primary" />}
              <span className="text-xs text-muted-foreground">{new Date(n.createdAt).toLocaleString()}</span>
            </div>
          </button>
        ))}
      </Card>

      {hasMore && (
        <Button variant="outline" onClick={() => loadPage(offset)} disabled={loading} className="self-center">
          {loading ? t('page.loading') : t('page.loadMore')}
        </Button>
      )}
    </div>
  )
}
