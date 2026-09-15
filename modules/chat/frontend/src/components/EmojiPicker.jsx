import { useEffect, useRef, useState } from 'react'
import { Smile } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'

// A plain grid of common emojis - no new npm dependency, just enough to
// cover the request ("ada emoji2"). Click inserts into the message box via
// onPick and closes the popover.
const EMOJIS = [
  '😀', '😁', '😂', '🤣', '😊', '😍', '😘', '😉', '😎', '🤩',
  '🥳', '😇', '🙂', '🙃', '😴', '🤔', '😅', '😭', '😢', '😡',
  '🤯', '😱', '🥺', '😬', '🙄', '😏', '🤗', '🤫', '🤝', '👍',
  '👎', '👏', '🙏', '💪', '✌️', '🤞', '👌', '🫡', '👋', '🔥',
  '🎉', '✅', '❌', '❤️', '💙', '💚', '💯', '⭐', '☕', '🚀',
]

export default function EmojiPicker({ onPick }) {
  const [open, setOpen] = useState(false)
  const ref = useRef(null)

  useEffect(() => {
    const onClickOutside = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false) }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [])

  return (
    <div className="relative" ref={ref}>
      <Button type="button" variant="ghost" size="icon" onClick={() => setOpen((v) => !v)} title="Emoji">
        <Smile className="h-4 w-4" />
      </Button>
      {open && (
        <Card className="absolute bottom-full left-0 z-30 mb-2 w-64 p-2 shadow-xl">
          <div className="grid grid-cols-8 gap-0.5">
            {EMOJIS.map((e) => (
              <button
                key={e}
                type="button"
                onClick={() => { onPick(e); setOpen(false) }}
                className="rounded-md p-1 text-lg hover:bg-muted"
              >
                {e}
              </button>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}
