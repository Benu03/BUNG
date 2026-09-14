import { useState } from 'react'
import { ArrowLeft, Boxes, LogOut, LayoutGrid } from 'lucide-react'
import BoardList from './components/BoardList.jsx'
import Board from './components/Board.jsx'
import { Button } from './components/ui/button.jsx'

export default function App() {
  const [boardId, setBoardId] = useState(null)

  // Auth (login/logout/session) is owned entirely by app-maintenance (see
  // /nginx/auth-common.conf) - this module never implements its own.
  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    window.location.href = '/'
  }

  return (
    <div className="min-h-svh bg-muted/30">
      <header className="sticky top-0 z-10 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3.5">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/15">
              <Boxes className="h-4 w-4" />
            </div>
            <div>
              <div className="flex items-center gap-1.5 text-sm font-semibold leading-none">
                <LayoutGrid className="h-3.5 w-3.5" />
                Kanban
              </div>
              <div className="text-xs text-primary-foreground/70">Boards, columns &amp; cards</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" asChild className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <a href="/">
                <ArrowLeft />
                Portal
              </a>
            </Button>
            <Button size="sm" onClick={logout} className="border border-white/25 bg-white/10 text-primary-foreground shadow-none hover:bg-white/20">
              <LogOut />
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8">
        {boardId
          ? <Board boardId={boardId} onBack={() => setBoardId(null)} />
          : <BoardList onOpen={setBoardId} />}
      </main>
    </div>
  )
}
