// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /chat/api/ in this module.
const API_BASE = `${import.meta.env.BASE_URL}api/`

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }
  if (res.status === 204) return null
  return res.json()
}

export const api = {
  me: () => request('me'),
  isAdmin: () => request('is-admin'),
  listUsers: () => request('users'),

  listFriends: () => request('friends'),
  listFriendRequests: () => request('friend-requests'),
  sendFriendRequest: (username) => request('friend-requests', { method: 'POST', body: JSON.stringify({ username }) }),
  acceptFriendRequest: (id) => request(`friend-requests/${id}/accept`, { method: 'POST' }),
  declineFriendRequest: (id) => request(`friend-requests/${id}/decline`, { method: 'POST' }),

  listConversations: () => request('conversations'),
  startConversation: (userId) => request('conversations', { method: 'POST', body: JSON.stringify({ userId }) }),
  listMessages: (conversationId, limit = 50) => request(`conversations/${conversationId}/messages?limit=${limit}`),
  sendMessage: (conversationId, body) => request(`conversations/${conversationId}/messages`, { method: 'POST', body: JSON.stringify({ body }) }),
  markRead: (conversationId) => request(`conversations/${conversationId}/read`, { method: 'POST' }),

  // Multipart, so this bypasses the JSON `request()` helper.
  uploadAttachment: async (conversationId, file) => {
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(`${API_BASE}conversations/${conversationId}/attachments`, { method: 'POST', body: form })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error(body.error || `Request failed: ${res.status}`)
    }
    return res.json()
  },
  // Not fetched via JS - used directly as an <a href> so the browser
  // handles the Content-Disposition download.
  downloadUrl: (messageId) => `${API_BASE}messages/${messageId}/download`,

  listBroadcasts: () => request('broadcasts'),
  createBroadcast: (body) => request('broadcasts', { method: 'POST', body: JSON.stringify({ body }) }),
}

// Real-time updates: friend requests/accepts, new messages, broadcasts -
// all delivered over one connection per user (see backend/hub.go), not
// scoped to whichever conversation happens to be open.
export function connectChatSocket(onMessage) {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${proto}://${window.location.host}${import.meta.env.BASE_URL}api/ws`
  let socket
  let closedByCaller = false
  let retryDelay = 1000

  const connect = () => {
    socket = new WebSocket(url)
    socket.onmessage = (evt) => {
      try {
        onMessage(JSON.parse(evt.data))
      } catch {
        // ignore malformed frames (e.g. server ping text)
      }
    }
    socket.onclose = () => {
      if (closedByCaller) return
      setTimeout(connect, retryDelay)
      retryDelay = Math.min(retryDelay * 2, 15000)
    }
    socket.onopen = () => { retryDelay = 1000 }
  }
  connect()

  return () => { closedByCaller = true; socket?.close() }
}
