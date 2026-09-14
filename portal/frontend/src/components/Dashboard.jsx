import { ArrowRight, Boxes, LogOut, PackageOpen } from 'lucide-react'
import { Button } from './ui/button.jsx'
import { Card, CardHeader, CardTitle, CardDescription, CardFooter } from './ui/card.jsx'

export default function Dashboard({ user, onLogout }) {
  const modules = user.modules || []

  return (
    <div className="min-h-svh bg-muted/40">
      <header className="border-b bg-background">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Boxes className="h-4 w-4" />
            </div>
            <span className="font-semibold tracking-tight">BUNG</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-sm text-muted-foreground">{user.fullName || user.username}</span>
            <Button variant="outline" size="sm" onClick={onLogout}>
              <LogOut />
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-10">
        <div className="mb-8">
          <h1 className="text-2xl font-semibold tracking-tight">Your modules</h1>
          <p className="text-sm text-muted-foreground">Pick a module to get started.</p>
        </div>

        {modules.length === 0 ? (
          <Card className="flex flex-col items-center gap-2 border-dashed py-16 text-center">
            <PackageOpen className="h-8 w-8 text-muted-foreground" />
            <p className="font-medium">No modules assigned yet</p>
            <p className="max-w-xs text-sm text-muted-foreground">
              Ask an administrator to assign a role with module access to your account.
            </p>
          </Card>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {modules.map((m) => (
              <a key={m.code} href={`/${m.code}/`} className="group">
                <Card className="h-full transition-colors group-hover:border-foreground/30">
                  <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                      {m.name}
                      <ArrowRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground" />
                    </CardTitle>
                    {m.description && <CardDescription>{m.description}</CardDescription>}
                  </CardHeader>
                  <CardFooter>
                    <span className="text-xs text-muted-foreground">/{m.code}/</span>
                  </CardFooter>
                </Card>
              </a>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
