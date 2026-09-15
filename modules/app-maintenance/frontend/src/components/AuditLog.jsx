import { useEffect, useState } from 'react'
import { Search, TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'
import { useTranslation } from '../lib/i18n.jsx'

const PAGE_SIZE = 50

function actionVariant(action) {
  if (action.includes('delete') || action.includes('failed') || action.includes('blocked')) return 'destructive'
  if (action.includes('create') || action.includes('login')) return 'default'
  return 'secondary'
}

export default function AuditLog() {
  const { t } = useTranslation()
  const [entries, setEntries] = useState([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('')

  const loadPage = async (nextOffset) => {
    setLoading(true)
    setError('')
    try {
      const page = await api.listAuditLog(PAGE_SIZE, nextOffset)
      setEntries((prev) => (nextOffset === 0 ? page : [...prev, ...page]))
      setHasMore(page.length === PAGE_SIZE)
      setOffset(nextOffset + page.length)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadPage(0) }, [])

  // Filters only what's already loaded (client-side) - "Load more" still
  // fetches the next raw page from the server regardless of the filter.
  const q = filter.trim().toLowerCase()
  const filteredEntries = q
    ? entries.filter((e) => [e.actorUsername, e.moduleCode, e.action, e.entityType].some((v) => v?.toLowerCase().includes(q)))
    : entries

  return (
    <div className="flex flex-col gap-4">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <div className="flex items-center gap-2 border-b p-3">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('common.filterPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="h-8 max-w-xs border-0 shadow-none focus-visible:ring-0"
          />
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">{t('auditLog.time')}</th>
                <th className="px-4 py-3 font-medium">{t('auditLog.actor')}</th>
                <th className="px-4 py-3 font-medium">{t('auditLog.module')}</th>
                <th className="px-4 py-3 font-medium">{t('auditLog.action')}</th>
                <th className="px-4 py-3 font-medium">{t('auditLog.entity')}</th>
                <th className="px-4 py-3 font-medium">{t('auditLog.ip')}</th>
              </tr>
            </thead>
            <tbody>
              {filteredEntries.map((e) => (
                <tr key={e.id} className="border-b last:border-0 hover:bg-muted/40">
                  <td className="whitespace-nowrap px-4 py-2.5 text-muted-foreground">{new Date(e.occurredAt).toLocaleString()}</td>
                  <td className="px-4 py-2.5">{e.actorUsername || <span className="text-muted-foreground">-</span>}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{e.moduleCode}</td>
                  <td className="px-4 py-2.5"><Badge variant={actionVariant(e.action)}>{e.action}</Badge></td>
                  <td className="px-4 py-2.5 text-muted-foreground">
                    {e.entityType}{e.entityId ? ` #${e.entityId.slice(0, 8)}` : ''}
                  </td>
                  <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{e.ipAddress}</td>
                </tr>
              ))}
              {filteredEntries.length === 0 && !loading && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                    {entries.length === 0 ? t('auditLog.noActivity') : t('common.noMatches')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {hasMore && (
        <Button variant="outline" onClick={() => loadPage(offset)} disabled={loading} className="self-center">
          {loading ? t('auditLog.loading') : t('auditLog.loadMore')}
        </Button>
      )}
    </div>
  )
}
