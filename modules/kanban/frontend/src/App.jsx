import { useEffect, useState } from 'react'
import { ArrowLeft, LogOut, LayoutGrid } from 'lucide-react'
import BoardList from './components/BoardList.jsx'
import Board from './components/Board.jsx'
import Logo from './components/Logo.jsx'
import ThemeToggle from './components/ThemeToggle.jsx'
import { Button } from './components/ui/button.jsx'
import { api } from './api.js'

export default function App() {
  const [boardId, setBoardId] = useState(null)
  const [me, setMe] = useState(null)

  useEffect(() => { api.me().then(setMe).catch(() => {}) }, [])

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
            <Logo className="h-8 w-8" />
            <div>
              <div className="flex items-center gap-1.5 text-sm font-semibold leading-none">
                <LayoutGrid className="h-3.5 w-3.5" />
                Kanban
              </div>
              <div className="text-xs text-primary-foreground/70">Boards, columns &amp; cards</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground" />
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
          ? <Board boardId={boardId} onBack={() => setBoardId(null)} me={me} />
          : <BoardList onOpen={setBoardId} />}
      </main>
    </div>
  )
}
