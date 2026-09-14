import { ArrowLeft, FolderOpen, LogOut } from 'lucide-react'
import FileList from './components/FileList.jsx'
import Logo from './components/Logo.jsx'
import ThemeToggle from './components/ThemeToggle.jsx'
import { Button } from './components/ui/button.jsx'

export default function App() {
  // Auth (login/logout/session) is owned entirely by app-maintenance (see
  // /nginx/auth-common.conf) - this module never implements its own.
  const logout = async () => {
    await fetch('/app-maintenance/api/auth/logout', { method: 'POST' })
    window.location.href = '/'
  }

  return (
    <div className="min-h-svh bg-muted/30">
      <header className="sticky top-0 z-10 bg-gradient-to-r from-primary to-primary-dark text-primary-foreground shadow-md">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3.5">
          <div className="flex items-center gap-2.5">
            <Logo className="h-8 w-8" />
            <div>
              <div className="flex items-center gap-1.5 text-sm font-semibold leading-none">
                <FolderOpen className="h-3.5 w-3.5" />
                My Storage
              </div>
              <div className="text-xs text-primary-foreground/70">Your private files</div>
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

      <main className="mx-auto max-w-4xl px-4 py-8">
        <FileList />
      </main>
    </div>
  )
}
