import { NavLink, useNavigate } from "react-router-dom"
import {
  LayoutDashboard,
  Bot,
  Users,
  Flag,
  Tags,
  BarChart3,
  Activity,
  Github,
  ArrowLeft,
  Home,
} from "lucide-react"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import { UserMenu } from "@/components/layout/UserMenu"

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
        <NavLink
          to="/console"
          className="flex items-center gap-2 px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="size-3.5" />
          {t('nav.backToConsole')}
        </NavLink>
        <div className="flex items-center justify-between px-3 py-1">
          <NavLink
            to="/"
            className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Home className="size-3.5" />
            {t('nav.backToHome')}
          </NavLink>
          <div className="flex items-center gap-1.5">
            <a
              href="https://github.com/peerclaw/peerclaw"
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground transition-colors"
              title={t('nav.github')}
            >
              <Github className="size-3.5" />
            </a>
            <LanguageSwitcher />
          </div>
        </div>
        {user && (
          <UserMenu
            user={user}
            onLogout={handleLogout}
          />
        )}
      </div>
    </aside>
  )
}
