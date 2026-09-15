import { useEffect, useRef, useState } from 'react'
import { CalendarDays, FileText, Search, Ticket as TicketIcon, LayoutGrid, User as UserIcon } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'

// Cross-module by design, same pattern as NotificationBell: fetches
// straight from /search/api/ (a different module's gateway path) and is
// duplicated verbatim into every frontend's header (portal,
// app-maintenance, kanban, my-storage, calendar, ticketing, notifications).
// The search gateway route doesn't require any specific module grant (see
// /modules/search/nginx.conf) - the backend itself scopes every result to
// what the caller can actually see.
const API_BASE = '/search/api/'
const DEBOUNCE_MS = 200
const MIN_QUERY_LEN = 2

const TYPE_ICON = {
  ticket: TicketIcon,
  event: CalendarDays,
  file: FileText,
  board: LayoutGrid,
  user: UserIcon,
}

async function fetchResults(q, signal) {
  const res = await fetch(`${API_BASE}search?q=${encodeURIComponent(q)}`, { signal })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return res.json()
}

export default function CommandPalette({ className }) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef(null)

  // Global Ctrl+K / Cmd+K toggle, from anywhere on the page (not just
  // while the trigger button has focus).
  useEffect(() => {
    const onKeyDown = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen((v) => !v)
      } else if (e.key === 'Escape') {
        setOpen(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 0)
    } else {
      setQuery('')
      setResults([])
      setActiveIndex(0)
    }
  }, [open])

  useEffect(() => {
    const q = query.trim()
    if (q.length < MIN_QUERY_LEN) {
      setResults([])
      setLoading(false)
      return
    }
    setLoading(true)
    const controller = new AbortController()
    const t = setTimeout(() => {
      fetchResults(q, controller.signal)
        .then((rs) => { setResults(rs); setActiveIndex(0) })
        .catch(() => {})
        .finally(() => setLoading(false))
    }, DEBOUNCE_MS)
    return () => { clearTimeout(t); controller.abort() }
  }, [query])

  const go = (r) => { window.location.href = r.link }

  const onKeyDown = (e) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((i) => Math.min(i + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter' && results[activeIndex]) {
      go(results[activeIndex])
    }
  }

  return (
    <>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setOpen(true)}
        title="Search (Ctrl+K)"
        className={className}
      >
        <Search className="h-4 w-4" />
      </Button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/50 px-4 pt-24" onClick={() => setOpen(false)}>
          <Card className="w-full max-w-lg overflow-hidden p-0 shadow-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-2 border-b px-3 py-2.5">
              <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
              <Input
                ref={inputRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={onKeyDown}
                placeholder="Search tickets, events, files, boards, people..."
                className="h-8 flex-1 border-0 shadow-none focus-visible:ring-0"
              />
              <kbd className="shrink-0 rounded border px-1.5 py-0.5 text-[10px] text-muted-foreground">Esc</kbd>
            </div>

            <div className="max-h-80 overflow-y-auto">
              {loading && (
                <p className="px-3 py-6 text-center text-xs text-muted-foreground">Searching...</p>
              )}
              {!loading && query.trim().length >= MIN_QUERY_LEN && results.length === 0 && (
                <p className="px-3 py-6 text-center text-xs text-muted-foreground">No results</p>
              )}
              {!loading && query.trim().length < MIN_QUERY_LEN && (
                <p className="px-3 py-6 text-center text-xs text-muted-foreground">Keep typing to search...</p>
              )}
              {results.map((r, i) => {
                const Icon = TYPE_ICON[r.type] || Search
                return (
                  <button
                    key={`${r.type}-${r.id}`}
                    onClick={() => go(r)}
                    onMouseEnter={() => setActiveIndex(i)}
                    className={`flex w-full items-center gap-2.5 border-b px-3 py-2.5 text-left last:border-0 ${
                      i === activeIndex ? 'bg-muted/70' : 'hover:bg-muted/40'
                    }`}
                  >
                    <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate text-sm font-medium">{r.title}</span>
                        <span className="shrink-0 text-[10px] uppercase tracking-wide text-muted-foreground">{r.moduleCode}</span>
                      </div>
                      {r.subtitle && <div className="truncate text-xs text-muted-foreground">{r.subtitle}</div>}
                    </div>
                  </button>
                )
              })}
            </div>
          </Card>
        </div>
      )}
    </>
  )
}
