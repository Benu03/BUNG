import { useEffect, useRef, useState } from 'react'
import { Pause, Play, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'

// Different gateway path than the rest of app-maintenance (its own api.js
// points at /app-maintenance/api/) - this hits the separate logs-backend
// service directly, gated the same way (app-maintenance's own module
// grant, see /modules/logs/nginx.conf) even though the path differs.
const API_BASE = '/logs/api/'
const MAX_LINES = 2000

export default function Logs() {
  const { t } = useTranslation()
  const [containers, setContainers] = useState([])
  const [selected, setSelected] = useState('')
  const [lines, setLines] = useState([])
  const [connected, setConnected] = useState(false)
  const [paused, setPaused] = useState(false)
  const [error, setError] = useState('')
  const scrollRef = useRef(null)

  const loadContainers = () => {
    fetch(`${API_BASE}containers`)
      .then((r) => r.json())
      .then((cs) => {
        setContainers(cs)
        setSelected((prev) => prev || cs[0]?.id || '')
      })
      .catch((e) => setError(e.message))
  }

  useEffect(() => { loadContainers() }, [])

  useEffect(() => {
    if (!selected || paused) {
      setConnected(false)
      return
    }
    setLines([])
    setError('')
    const es = new EventSource(`${API_BASE}containers/${selected}/stream?tail=200`)
    setConnected(true)
    es.onmessage = (e) => {
      setLines((prev) => {
        const next = [...prev, e.data]
        return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next
      })
    }
    es.onerror = () => setConnected(false)
    return () => {
      es.close()
      setConnected(false)
    }
  }, [selected, paused])

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight
  }, [lines])

  return (
    <div className="flex flex-col gap-4">
      {error && (
        <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-2 text-sm"
        >
          {containers.map((c) => (
            <option key={c.id} value={c.id}>{c.name} ({c.state})</option>
          ))}
        </select>
        <Button variant="outline" size="sm" onClick={loadContainers} title={t('logs.refresh')}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
        <Button variant="outline" size="sm" onClick={() => setPaused((p) => !p)}>
          {paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
          {paused ? t('logs.resume') : t('logs.pause')}
        </Button>
        <Button variant="outline" size="sm" onClick={() => setLines([])}>
          <Trash2 className="h-3.5 w-3.5" />
          {t('logs.clear')}
        </Button>
        <span className={`flex items-center gap-1.5 text-xs ${connected ? 'text-emerald-600' : 'text-muted-foreground'}`}>
          <span className={`h-1.5 w-1.5 rounded-full ${connected ? 'bg-emerald-500' : 'bg-muted-foreground/40'}`} />
          {connected ? t('logs.live') : t('logs.disconnected')}
        </span>
      </div>

      <Card className="overflow-hidden p-0">
        <pre ref={scrollRef} className="max-h-[32rem] overflow-y-auto bg-slate-950 p-3 text-xs leading-relaxed text-slate-100">
          {lines.length === 0 ? (
            <span className="text-slate-500">{t('logs.empty')}</span>
          ) : (
            lines.map((l, i) => <div key={i} className="whitespace-pre-wrap break-all">{l}</div>)
          )}
        </pre>
      </Card>

      <p className="text-center text-xs text-muted-foreground">{t('logs.footerHint')}</p>
    </div>
  )
}
