import { NavLink } from "react-router-dom"
import { LayoutDashboard, Bot } from "lucide-react"

const links = [
  { to: "/admin", label: "Overview", icon: LayoutDashboard },
  { to: "/admin/agents", label: "Agents", icon: Bot },
]

export function Sidebar() {
  return (
    <aside className="flex h-screen w-56 flex-col border-r border-border bg-card">
      <div className="flex h-14 items-center gap-2 border-b border-border px-4">
        <div className="size-7 rounded-md bg-primary flex items-center justify-center">
          <span className="text-xs font-bold text-primary-foreground">PC</span>
        </div>
        <span className="font-semibold text-sm">PeerClaw Admin</span>
      </div>

      <nav className="flex-1 space-y-1 p-3">
        {links.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === "/admin"}
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

      <div className="border-t border-border p-3">
        <NavLink to="/" className="text-xs text-muted-foreground hover:text-foreground">
          Back to Public Site
        </NavLink>
      </div>
    </aside>
  )
}
