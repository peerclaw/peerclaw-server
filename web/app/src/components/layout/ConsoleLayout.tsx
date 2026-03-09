import { NavLink, Outlet, useNavigate } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import {
  LayoutDashboard,
  Bot,
  PlusCircle,
  Activity,
  KeyRound,
  LogOut,
  Shield,
} from "lucide-react"

const navLinks = [
  { to: "/console", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/console/agents", label: "My Agents", icon: Bot, end: false },
  { to: "/console/publish", label: "Publish Agent", icon: PlusCircle, end: false },
  { to: "/console/invocations", label: "Invocations", icon: Activity, end: false },
  { to: "/console/api-keys", label: "API Keys", icon: KeyRound, end: false },
]

export function ConsoleLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Sidebar */}
      <aside className="flex h-screen w-60 flex-col border-r border-border bg-card">
        {/* Logo */}
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-md object-cover" />
          <span className="font-semibold text-sm">PeerClaw Console</span>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 p-3">
          {navLinks.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                  isActive
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                }`
              }
            >
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* User / Sign Out */}
        <div className="border-t border-border p-3 space-y-2">
          {user && (
            <div className="px-3 py-1">
              <p className="text-sm font-medium text-foreground truncate">
                {user.display_name || user.email}
              </p>
              <p className="text-xs text-muted-foreground truncate">{user.email}</p>
            </div>
          )}
          {user?.role === "admin" && (
            <NavLink
              to="/admin"
              className={({ isActive }) =>
                `flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                  isActive
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                }`
              }
            >
              <Shield className="size-4" />
              Admin Panel
            </NavLink>
          )}
          <button
            onClick={handleLogout}
            className="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
          >
            <LogOut className="size-4" />
            Sign out
          </button>
          <NavLink
            to="/"
            className="block text-xs text-muted-foreground hover:text-foreground px-3"
          >
            Back to Public Site
          </NavLink>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
