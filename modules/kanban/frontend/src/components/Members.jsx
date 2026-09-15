import { useEffect, useState } from 'react'
import { Crown, Trash2, X } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'
import UserPicker from './UserPicker.jsx'

export default function Members({ board, isOwner, onClose, onChanged }) {
  const { t } = useTranslation()
  const [members, setMembers] = useState([])
  const [allUsers, setAllUsers] = useState([])
  const [error, setError] = useState('')

  const load = () => api.listMembers(board.id).then(setMembers).catch((e) => setError(e.message))

  useEffect(() => { load() }, [board.id])
  useEffect(() => { api.listUsers().then(setAllUsers).catch(() => {}) }, [])

  const invite = async (user) => {
    setError('')
    try {
      await api.addMember(board.id, user.username)
      load()
      onChanged?.()
    } catch (e) {
      setError(e.message)
    }
  }

  const remove = async (userId) => {
    if (!confirm(t('members.confirmRemove'))) return
    try {
      await api.removeMember(board.id, userId)
      load()
      onChanged?.()
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className="fixed inset-0 z-20 flex items-start justify-center bg-black/40 p-4 pt-20" onClick={onClose}>
      <Card className="w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{t('members.title')}</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose}><X className="h-4 w-4" /></Button>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {error && (
            <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
          )}

          <div className="flex flex-col gap-1.5">
            {members.map((m) => (
              <div key={m.userId} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                <div className="flex items-center gap-2">
                  {m.role === 'owner' && <Crown className="h-3.5 w-3.5 text-amber-500" />}
                  <span className="font-medium">{m.fullName || m.username}</span>
                  <span className="text-xs text-muted-foreground">@{m.username}</span>
                </div>
                {isOwner && m.role !== 'owner' && (
                  <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => remove(m.userId)}>
                    <Trash2 className="h-3.5 w-3.5 text-destructive" />
                  </Button>
                )}
              </div>
            ))}
          </div>

          {isOwner && (
            <UserPicker
              users={allUsers}
              excludeIds={members.map((m) => m.userId)}
              onPick={invite}
              placeholder={t('members.inviteUsername')}
            />
          )}
          {!isOwner && (
            <p className="text-xs text-muted-foreground">{t('members.ownerOnlyHint')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
