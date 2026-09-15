import { useEffect, useRef, useState } from 'react'
import { Input } from './ui/input.jsx'

// A username text box with a live-filtered dropdown of matching users
// (by username or full name), instead of requiring the exact username to
// be typed. Clicking a suggestion picks it and clears the box. `users` is
// the full directory (fetched once by the caller); `excludeIds` hides
// people already on the list (e.g. existing board members) from the
// suggestions.
export default function UserPicker({ users, excludeIds = [], onPick, placeholder }) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const ref = useRef(null)

  useEffect(() => {
    const onClickOutside = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false) }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [])

  const q = query.trim().toLowerCase()
  const matches = q
    ? users
        .filter((u) => !excludeIds.includes(u.id))
        .filter((u) => [u.username, u.fullName].some((v) => v?.toLowerCase().includes(q)))
        .slice(0, 8)
    : []

  const pick = (u) => {
    onPick(u)
    setQuery('')
    setOpen(false)
  }

  return (
    <div className="relative flex-1" ref={ref}>
      <Input
        placeholder={placeholder}
        value={query}
        onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
        onFocus={() => setOpen(true)}
        className="h-9"
      />
      {open && matches.length > 0 && (
        <div className="absolute left-0 right-0 top-full z-30 mt-1 max-h-48 overflow-y-auto rounded-md border bg-popover shadow-md">
          {matches.map((u) => (
            <button
              key={u.id}
              type="button"
              onClick={() => pick(u)}
              className="flex w-full flex-col items-start px-3 py-1.5 text-left text-sm hover:bg-muted"
            >
              <span className="font-medium">{u.fullName || u.username}</span>
              <span className="text-xs text-muted-foreground">@{u.username}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
