import { NavLink, useNavigate } from "react-router-dom"
import {
  LayoutDashboard,
  Bot,
  Users,
  Flag,
  Tags,
  BarChart3,
  Activity,
  LogOut,
  ExternalLink,
} from "lucide-react"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"

export function Sidebar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const links = [
    { to: "/admin", label: t('nav.overview'), icon: LayoutDashboard },
    { to: "/admin/users", label: t('nav.users'), icon: Users },
    { to: "/admin/agents", label: t('nav.agents'), icon: Bot },
    { to: "/admin/reports", label: t('nav.reports'), icon: Flag },
    { to: "/admin/categories", label: t('nav.categories'), icon: Tags },
    { to: "/admin/analytics", label: t('nav.analytics'), icon: BarChart3 },
    { to: "/admin/invocations", label: t('nav.invocations'), icon: Activity },
  ]

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <aside className="flex h-screen w-56 flex-col border-r border-border bg-card">
      <div className="flex h-14 items-center gap-2 border-b border-border px-4">
        <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-md object-cover" />
        <span className="font-semibold text-sm">{t('nav.peerclawAdmin')}</span>
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

      <div className="border-t border-border p-3 space-y-2">
        {user && (
          <div className="px-3 py-1">
            <p className="text-xs font-medium truncate">{user.display_name || user.email}</p>
            <p className="text-xs text-muted-foreground truncate">{user.email}</p>
          </div>
        )}
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors"
        >
          <LogOut className="size-4" />
          {t('nav.logout')}
        </button>
        <NavLink
          to="/"
          className="flex items-center gap-2.5 rounded-md px-3 py-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <ExternalLink className="size-3.5" />
          {t('nav.backToPublicSite')}
        </NavLink>
        <div className="px-3 pt-1">
          <LanguageSwitcher />
        </div>
      </div>
    </aside>
  )
}
