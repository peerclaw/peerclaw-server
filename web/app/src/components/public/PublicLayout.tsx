import { Outlet, Link, NavLink } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"

export function PublicLayout() {
  const { user, logout } = useAuth()

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <div className="size-7 rounded-md bg-primary flex items-center justify-center">
              <span className="text-xs font-bold text-primary-foreground">PC</span>
            </div>
            <span className="font-semibold text-sm">PeerClaw</span>
          </Link>

          <nav className="flex items-center gap-4">
            <NavLink
              to="/directory"
              className={({ isActive }) =>
                `text-sm transition-colors ${
                  isActive
                    ? "text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`
              }
            >
              Directory
            </NavLink>
            <NavLink
              to="/playground"
              className={({ isActive }) =>
                `text-sm transition-colors ${
                  isActive
                    ? "text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`
              }
            >
              Playground
            </NavLink>
            {user ? (
              <>
                <NavLink
                  to="/console"
                  className={({ isActive }) =>
                    `text-sm transition-colors ${
                      isActive
                        ? "text-foreground font-medium"
                        : "text-muted-foreground hover:text-foreground"
                    }`
                  }
                >
                  Console
                </NavLink>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    {user.display_name || user.email}
                  </span>
                  <button
                    onClick={() => logout()}
                    className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  >
                    Sign Out
                  </button>
                </div>
              </>
            ) : (
              <>
                <Link
                  to="/admin"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  Admin
                </Link>
                <Link
                  to="/login"
                  className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                >
                  Sign In
                </Link>
              </>
            )}
          </nav>
        </div>
      </header>

      <main>
        <Outlet />
      </main>
    </div>
  )
}
