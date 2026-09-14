import { useEffect, useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { Badge } from './ui/badge.jsx'

const PAGE_SIZE = 50

function actionVariant(action) {
  if (action.includes('delete') || action.includes('failed') || action.includes('blocked')) return 'destructive'
  if (action.includes('create') || action.includes('login')) return 'default'
  return 'secondary'
}

export default function AuditLog() {
  const [entries, setEntries] = useState([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

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

  return (
    <div className="flex flex-col gap-4">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">Time</th>
                <th className="px-4 py-3 font-medium">Actor</th>
                <th className="px-4 py-3 font-medium">Module</th>
                <th className="px-4 py-3 font-medium">Action</th>
                <th className="px-4 py-3 font-medium">Entity</th>
                <th className="px-4 py-3 font-medium">IP</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
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
              {entries.length === 0 && !loading && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">No activity yet.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {hasMore && (
        <Button variant="outline" onClick={() => loadPage(offset)} disabled={loading} className="self-center">
          {loading ? 'Loading...' : 'Load more'}
        </Button>
      )}
    </div>
  )
}
