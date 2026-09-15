import { useCallback, useEffect, useRef, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import ConversationSidebar from './ConversationSidebar.jsx'
import MessageThread from './MessageThread.jsx'
import BroadcastsPanel from './BroadcastsPanel.jsx'
import { Button } from './ui/button.jsx'
import { api, connectChatSocket } from '../api.js'

// Orchestrates the WhatsApp-Web-style two-panel layout: a conversation/
// friends sidebar on the left, and either a message thread or the
// broadcasts feed on the right. All state lives here; the panels below are
// mostly dumb renderers driven by props + callbacks.
export default function ChatPage({ me }) {
  const [users, setUsers] = useState([])
  const [friends, setFriends] = useState([])
  const [friendRequests, setFriendRequests] = useState([])
  const [conversations, setConversations] = useState([])
  const [isAdmin, setIsAdmin] = useState(false)

  const [view, setView] = useState('chat') // 'chat' | 'broadcasts'
  const [selectedConversationId, setSelectedConversationId] = useState(null)
  const [messages, setMessages] = useState([])
  const [messagesLoading, setMessagesLoading] = useState(false)
  const [broadcasts, setBroadcasts] = useState([])
  const [hasNewBroadcast, setHasNewBroadcast] = useState(false)

  const selectedConversationRef = useRef(null)
  selectedConversationRef.current = view === 'chat' ? selectedConversationId : null

  // NOTE: these must actually return their promises - callers `await` them
  // (e.g. startConversation waits for the fresh list before opening the
  // thread). An earlier version of this function had no `return`, so the
  // `await` resolved immediately without waiting for the fetch, racing
  // against the just-created conversation not being in state yet - showed
  // up as a blank thread the first time you started a new personal chat.
  const refreshFriends = useCallback(() => {
    return Promise.all([
      api.listFriends().then(setFriends).catch(() => {}),
      api.listFriendRequests().then(setFriendRequests).catch(() => {}),
    ])
  }, [])

  const refreshConversations = useCallback(() => {
    return api.listConversations().then(setConversations).catch(() => {})
  }, [])

  useEffect(() => {
    api.listUsers().then(setUsers).catch(() => {})
    api.isAdmin().then((d) => setIsAdmin(!!d.isAdmin)).catch(() => {})
    refreshFriends()
    refreshConversations()
  }, [refreshFriends, refreshConversations])

  // WebSocket: one connection per user, delivering friend/message/broadcast
  // events regardless of which conversation is currently open (see
  // backend/hub.go - keyed by user id, not by conversation).
  useEffect(() => {
    if (!me?.id) return
    const disconnect = connectChatSocket((evt) => {
      if (evt.type === 'friend_request' || evt.type === 'friend_accepted') {
        refreshFriends()
        refreshConversations()
        return
      }
      if (evt.type === 'broadcast') {
        setBroadcasts((prev) => [evt.broadcast, ...prev])
        setHasNewBroadcast(true)
        return
      }
      if (evt.type === 'message') {
        const msg = evt.message
        const isOpen = evt.conversationId === selectedConversationRef.current
        if (isOpen) {
          setMessages((prev) => (prev.some((m) => m.id === msg.id) ? prev : [...prev, msg]))
          if (msg.senderId !== me.id) api.markRead(evt.conversationId).catch(() => {})
        }
        setConversations((prev) => {
          const idx = prev.findIndex((c) => c.id === evt.conversationId)
          if (idx === -1) {
            refreshConversations()
            return prev
          }
          const next = [...prev]
          const conv = { ...next[idx], lastMessage: msg }
          if (!isOpen && msg.senderId !== me.id) conv.unreadCount = (conv.unreadCount || 0) + 1
          else conv.unreadCount = 0
          next.splice(idx, 1)
          next.unshift(conv)
          return next
        })
      }
    })
    return disconnect
  }, [me?.id, refreshFriends, refreshConversations])

  const openConversation = async (id) => {
    setView('chat')
    setSelectedConversationId(id)
    setMessagesLoading(true)
    try {
      const list = await api.listMessages(id)
      setMessages(list)
      await api.markRead(id)
      setConversations((prev) => prev.map((c) => (c.id === id ? { ...c, unreadCount: 0 } : c)))
    } catch {
      setMessages([])
    } finally {
      setMessagesLoading(false)
    }
  }

  const startConversation = async (userId) => {
    const { id } = await api.startConversation(userId)
    // Insert the new conversation into state immediately from data we
    // already have (the friends list), rather than relying solely on a
    // refetch racing against openConversation below.
    setConversations((prev) => {
      if (prev.some((c) => c.id === id)) return prev
      const friend = friends.find((f) => f.id === userId) || { id: userId, username: '', fullName: '' }
      return [{ id, friend, lastMessage: null, unreadCount: 0, createdAt: new Date().toISOString() }, ...prev]
    })
    refreshConversations() // reconcile with the server in the background
    await openConversation(id)
  }

  const sendMessage = async (body) => {
    const msg = await api.sendMessage(selectedConversationId, body)
    setMessages((prev) => [...prev, msg])
    setConversations((prev) => {
      const idx = prev.findIndex((c) => c.id === selectedConversationId)
      if (idx === -1) return prev
      const next = [...prev]
      const conv = { ...next[idx], lastMessage: msg, unreadCount: 0 }
      next.splice(idx, 1)
      next.unshift(conv)
      return next
    })
  }

  const uploadAttachment = async (file) => {
    const msg = await api.uploadAttachment(selectedConversationId, file)
    setMessages((prev) => [...prev, msg])
    setConversations((prev) => {
      const idx = prev.findIndex((c) => c.id === selectedConversationId)
      if (idx === -1) return prev
      const next = [...prev]
      const conv = { ...next[idx], lastMessage: msg, unreadCount: 0 }
      next.splice(idx, 1)
      next.unshift(conv)
      return next
    })
  }

  const addFriend = async (username) => {
    await api.sendFriendRequest(username).catch(() => {})
    refreshFriends()
  }

  const acceptRequest = async (id) => {
    await api.acceptFriendRequest(id)
    refreshFriends()
    refreshConversations()
  }

  const declineRequest = async (id) => {
    await api.declineFriendRequest(id)
    refreshFriends()
  }

  const openBroadcasts = () => {
    setView('broadcasts')
    setHasNewBroadcast(false)
    api.listBroadcasts().then(setBroadcasts).catch(() => {})
  }

  const sendBroadcast = async (body) => {
    await api.createBroadcast(body)
  }

  const selectedConversation = conversations.find((c) => c.id === selectedConversationId) || null
  const showThreadOnMobile = view === 'broadcasts' || !!selectedConversationId

  return (
    <div className="flex h-full min-h-0">
      <div className={`min-h-0 ${showThreadOnMobile ? 'hidden sm:flex' : 'flex'}`}>
        <ConversationSidebar
          me={me}
          users={users}
          friends={friends}
          friendRequests={friendRequests}
          conversations={conversations}
          selectedConversationId={selectedConversationId}
          view={view}
          hasNewBroadcast={hasNewBroadcast}
          isAdmin={isAdmin}
          onSelectConversation={openConversation}
          onOpenBroadcasts={openBroadcasts}
          onAddFriend={addFriend}
          onAcceptRequest={acceptRequest}
          onDeclineRequest={declineRequest}
          onStartConversation={startConversation}
        />
      </div>

      <div className={`min-h-0 flex-1 ${showThreadOnMobile ? 'flex' : 'hidden sm:flex'} flex-col`}>
        <div className="shrink-0 border-b bg-card px-2 py-1.5 sm:hidden">
          <Button variant="ghost" size="sm" onClick={() => { setSelectedConversationId(null); setView('chat') }}>
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
        </div>
        <div className="min-h-0 flex-1">
          {view === 'broadcasts' ? (
            <BroadcastsPanel isAdmin={isAdmin} broadcasts={broadcasts} onSend={sendBroadcast} />
          ) : (
            <MessageThread
              me={me}
              conversation={selectedConversation}
              messages={messages}
              loading={messagesLoading}
              onSend={sendMessage}
              onUpload={uploadAttachment}
            />
          )}
        </div>
      </div>
    </div>
  )
}
