import { useState } from 'react'
import { Check, Megaphone, UserPlus, X } from 'lucide-react'
import { Button } from './ui/button.jsx'
import UserPicker from './UserPicker.jsx'
import { useTranslation } from '../lib/i18n.jsx'

function formatTime(iso) {
  const d = new Date(iso)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { day: '2-digit', month: 'short' })
}

function Avatar({ name }) {
  return (
    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-primary-dark text-sm font-semibold text-primary-foreground">
      {(name || '?').slice(0, 1).toUpperCase()}
    </div>
  )
}

export default function ConversationSidebar({
  me,
  users,
  friends,
  friendRequests,
  conversations,
  selectedConversationId,
  view,
  hasNewBroadcast,
  isAdmin,
  onSelectConversation,
  onOpenBroadcasts,
  onAddFriend,
  onAcceptRequest,
  onDeclineRequest,
  onStartConversation,
}) {
  const { t } = useTranslation()
  const [showAddFriend, setShowAddFriend] = useState(false)

  const incoming = friendRequests.filter((r) => r.direction === 'incoming')
  const busyUserIds = new Set([
    me?.id,
    ...friends.map((f) => f.id),
    ...friendRequests.map((r) => (r.direction === 'incoming' ? r.fromUser.id : r.toUser.id)),
  ])
  const conversedFriendIds = new Set(conversations.map((c) => c.friend.id))
  const friendsWithoutChat = friends.filter((f) => !conversedFriendIds.has(f.id))

  return (
    <div className="flex h-full min-h-0 w-full flex-col border-r bg-card sm:w-80">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b px-3 py-3">
        <span className="text-sm font-semibold">{t('chat.title')}</span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            title={t('chat.addFriend')}
            onClick={() => setShowAddFriend((v) => !v)}
            className={showAddFriend ? 'bg-accent' : ''}
          >
            <UserPlus className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={t('chat.broadcasts')}
            onClick={onOpenBroadcasts}
            className={view === 'broadcasts' ? 'bg-accent' : ''}
          >
            <span className="relative">
              <Megaphone className="h-4 w-4" />
              {hasNewBroadcast && <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-destructive" />}
            </span>
          </Button>
        </div>
      </div>

      {showAddFriend && (
        <div className="shrink-0 border-b px-3 py-2.5">
          <UserPicker
            users={users}
            excludeIds={[...busyUserIds]}
            placeholder={t('chat.addFriendPlaceholder')}
            onPick={(u) => { onAddFriend(u.username); setShowAddFriend(false) }}
          />
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto">
        {incoming.length > 0 && (
          <div className="border-b bg-muted/40 px-3 py-2">
            <div className="mb-1.5 text-xs font-semibold text-muted-foreground">{t('chat.pendingRequests')}</div>
            <div className="space-y-1.5">
              {incoming.map((r) => (
                <div key={r.id} className="flex items-center gap-2 rounded-lg bg-card px-2.5 py-2 shadow-sm">
                  <Avatar name={r.fromUser.fullName || r.fromUser.username} />
                  <span className="min-w-0 flex-1 truncate text-sm">{r.fromUser.fullName || r.fromUser.username}</span>
                  <Button size="icon" className="h-7 w-7" title={t('chat.accept')} onClick={() => onAcceptRequest(r.id)}>
                    <Check className="h-3.5 w-3.5" />
                  </Button>
                  <Button size="icon" variant="outline" className="h-7 w-7" title={t('chat.decline')} onClick={() => onDeclineRequest(r.id)}>
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}

        {conversations.length === 0 && friendsWithoutChat.length === 0 && (
          <p className="px-3 py-8 text-center text-xs text-muted-foreground">{t('chat.startChattingHint')}</p>
        )}

        {conversations.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelectConversation(c.id)}
            className={`flex w-full items-center gap-2.5 border-b px-3 py-2.5 text-left ${
              view === 'chat' && selectedConversationId === c.id ? 'bg-accent' : 'hover:bg-muted/50'
            }`}
          >
            <Avatar name={c.friend.fullName || c.friend.username} />
            <div className="min-w-0 flex-1">
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-sm font-medium">{c.friend.fullName || c.friend.username}</span>
                {c.lastMessage && <span className="shrink-0 text-[10px] text-muted-foreground">{formatTime(c.lastMessage.createdAt)}</span>}
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-xs text-muted-foreground">
                  {c.lastMessage ? (c.lastMessage.body || `📎 ${c.lastMessage.attachmentFilename}`) : t('chat.noMessages')}
                </span>
                {c.unreadCount > 0 && (
                  <span className="flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground">
                    {c.unreadCount > 9 ? '9+' : c.unreadCount}
                  </span>
                )}
              </div>
            </div>
          </button>
        ))}

        {friendsWithoutChat.length > 0 && (
          <div>
            <div className="border-y bg-muted/40 px-3 py-1.5 text-xs font-semibold text-muted-foreground">{t('chat.friendsNoChat')}</div>
            {friendsWithoutChat.map((f) => (
              <button
                key={f.id}
                onClick={() => onStartConversation(f.id)}
                className="flex w-full items-center gap-2.5 border-b px-3 py-2.5 text-left hover:bg-muted/50"
              >
                <Avatar name={f.fullName || f.username} />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{f.fullName || f.username}</div>
                  <div className="truncate text-xs text-muted-foreground">@{f.username}</div>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
