import { NavLink, Outlet, useNavigate } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import { UserMenu } from "@/components/layout/UserMenu"
import {
  LayoutDashboard,
  Bot,
  PlusCircle,
  Activity,
  KeyRound,
  Github,
  Lock,
  Home,
} from "lucide-react"

export function ConsoleLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const navLinks = [
    { to: "/console", label: t('nav.dashboard'), icon: LayoutDashboard, end: true },
    { to: "/console/agents", label: t('nav.myAgents'), icon: Bot, end: false },
    { to: "/console/register", label: t('nav.registerAgent'), icon: PlusCircle, end: false },
    { to: "/console/invocations", label: t('nav.invocations'), icon: Activity, end: false },
    { to: "/console/access-requests", label: t('nav.accessRequests'), icon: Lock, end: false },
    { to: "/console/api-keys", label: t('nav.apiKeys'), icon: KeyRound, end: false },
  ]

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Sidebar */}
      <aside className="flex h-screen w-60 flex-col border-r border-border bg-card">
        {/* Logo + utility icons */}
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          <div className="flex items-center gap-2">
            <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-md object-cover" />
            <span className="font-semibold text-sm">{t('nav.peerclawConsole')}</span>
          </div>
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

        {/* Bottom: back link then user menu */}
        <div className="border-t border-border p-3 space-y-2">
          <NavLink
            to="/"
            className="flex items-center gap-2 px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Home className="size-3.5" />
            {t('nav.backToHome')}
          </NavLink>
          {user && (
            <UserMenu
              user={user}
              onLogout={handleLogout}
              showAdminLink={user.role === "admin"}
            />
          )}
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
