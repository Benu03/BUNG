import { useEffect, useState } from 'react'
import { MessageCircle } from 'lucide-react'
import { Button } from './ui/button.jsx'

// Cross-module by design, same pattern as NotificationBell: polls
// /chat/api/ (a different module's gateway path) for the caller's total
// unread message count across all conversations, and links straight to
// the chat module. Duplicated verbatim into every frontend's header
// (portal, app-maintenance, kanban, my-storage, calendar, ticketing,
// notifications). Unlike NotificationBell/CommandPalette, chat IS a
// module grant like any other (see /modules/chat/nginx.conf) - so a user
// without it would otherwise see a dead icon that redirects to login on
// click. Renders nothing until the first successful fetch confirms access,
// and hides itself again if a 403 ever comes back.
const API_BASE = '/chat/api/'
const POLL_MS = 30000

async function fetchUnread() {
  const res = await fetch(`${API_BASE}conversations`)
  if (res.status === 403) return null
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  const conversations = await res.json()
  return conversations.reduce((sum, c) => sum + (c.unreadCount || 0), 0)
}

export default function ChatButton({ className }) {
  const [unread, setUnread] = useState(0)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const load = () => fetchUnread()
      .then((count) => {
        if (count === null) { setVisible(false); return }
        setVisible(true)
        setUnread(count)
      })
      .catch(() => {})
    load()
    const id = setInterval(load, POLL_MS)
    return () => clearInterval(id)
  }, [])

  if (!visible) return null

  return (
    <Button variant="ghost" size="icon" asChild title="Chat" className={className}>
      <a href="/chat/">
        <span className="relative">
          <MessageCircle className="h-4 w-4" />
          {unread > 0 && (
            <span className="absolute -right-1.5 -top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold text-destructive-foreground">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </span>
      </a>
    </Button>
  )
}
