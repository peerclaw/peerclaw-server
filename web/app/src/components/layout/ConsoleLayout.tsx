import { useState, useEffect } from "react"
import { NavLink, Outlet, useNavigate, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"
import { useIsMobile } from "@/hooks/use-mobile"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import { UserMenu } from "@/components/layout/UserMenu"
import { NotificationBell } from "@/components/layout/NotificationBell"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip"
import {
  LayoutDashboard,
  Bot,
  Search,
  Activity,
  KeyRound,
  Github,
  Lock,
  Home,
  Bell,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react"

const COLLAPSED_KEY = "peerclaw_sidebar_collapsed"

export function ConsoleLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(() => {
    try { return localStorage.getItem(COLLAPSED_KEY) === "true" } catch { return false }
  })

  // Close mobile sheet on route change
  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  const toggleCollapsed = () => {
    setCollapsed((prev) => {
      const next = !prev
      try { localStorage.setItem(COLLAPSED_KEY, String(next)) } catch {}
      return next
    })
  }

  const navLinks = [
    { to: "/console", label: t('nav.dashboard'), icon: LayoutDashboard, end: true },
    { to: "/console/agents", label: t('nav.myAgents'), icon: Bot, end: false },
    { to: "/console/discover", label: t('nav.discoverAgents'), icon: Search, end: false },
    { to: "/console/invocations", label: t('nav.invocations'), icon: Activity, end: false },
    { to: "/console/access-requests", label: t('nav.accessRequests'), icon: Lock, end: false },
    { to: "/console/api-keys", label: t('nav.apiKeys'), icon: KeyRound, end: false },
    { to: "/console/notifications", label: t('nav.notifications'), icon: Bell, end: false },
  ]

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  const sidebarContent = (forMobile = false) => {
    const isCollapsed = !forMobile && collapsed

    return (
      <>
        {/* Logo */}
        <div className="flex h-14 items-center border-b border-border px-4">
          <div className="flex items-center gap-2">
            <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-md object-cover" />
            {!isCollapsed && (
              <span className="font-semibold text-sm">{t('nav.peerclawConsole')}</span>
            )}
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 p-3">
          {navLinks.map(({ to, label, icon: Icon, end }) => {
            const link = (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  `flex items-center ${isCollapsed ? "justify-center" : "gap-2.5"} rounded-md px-3 py-2 text-sm transition-colors ${
                    isActive
                      ? "bg-accent text-accent-foreground font-medium"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  }`
                }
              >
                <Icon className="size-4 shrink-0" />
                {!isCollapsed && label}
              </NavLink>
            )

            if (isCollapsed) {
              return (
                <Tooltip key={to}>
                  <TooltipTrigger asChild>{link}</TooltipTrigger>
                  <TooltipContent side="right">{label}</TooltipContent>
                </Tooltip>
              )
            }
            return link
          })}
        </nav>

        {/* Bottom */}
        <div className="border-t border-border p-3 space-y-2">
          {isCollapsed ? (
            <div className="flex flex-col items-center gap-2">
              <NavLink
                to="/"
                className="text-muted-foreground hover:text-foreground transition-colors"
                title={t('nav.backToHome')}
              >
                <Home className="size-3.5" />
              </NavLink>
              <a
                href="https://github.com/peerclaw/peerclaw"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground transition-colors"
                title={t('nav.github')}
              >
                <Github className="size-3.5" />
              </a>
            </div>
          ) : (
            <>
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
                  showAdminLink={user.role === "admin"}
                />
              )}
            </>
          )}
        </div>
      </>
    )
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:p-4 focus:bg-background focus:text-foreground">
        {t('common.skipToContent')}
      </a>
      {/* Desktop sidebar */}
      {!isMobile && (
        <aside
          className={`hidden md:flex h-screen flex-col border-r border-border bg-card transition-all duration-200 ${
            collapsed ? "w-14" : "w-60"
          }`}
        >
          {sidebarContent(false)}
        </aside>
      )}

      {/* Mobile sheet */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-60 p-0">
          <SheetTitle className="sr-only">{t('nav.menu')}</SheetTitle>
          <div className="flex h-full flex-col">
            {sidebarContent(true)}
          </div>
        </SheetContent>
      </Sheet>

      {/* Main content */}
      <main id="main-content" className="flex-1 overflow-y-auto">
        {/* Mobile header */}
        <div className={`flex h-12 items-center border-b border-border px-4 md:px-6 ${isMobile ? "" : "justify-end"}`}>
          {isMobile && (
            <button
              onClick={() => setMobileOpen(true)}
              className="mr-2 rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
              aria-label={t('nav.menu')}
            >
              <Menu className="size-5" />
            </button>
          )}
          {isMobile && (
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <img src="/logo.jpg" alt="PeerClaw" className="size-6 rounded-md object-cover" />
              <span className="font-semibold text-sm truncate">{t('nav.peerclawConsole')}</span>
            </div>
          )}
          {!isMobile && !collapsed && (
            <button
              onClick={toggleCollapsed}
              className="mr-auto rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
              title={t('nav.collapseSidebar')}
              aria-label={t('nav.toggleSidebar')}
            >
              <PanelLeftClose className="size-4" />
            </button>
          )}
          {!isMobile && collapsed && (
            <button
              onClick={toggleCollapsed}
              className="mr-auto rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
              title={t('nav.expandSidebar')}
              aria-label={t('nav.toggleSidebar')}
            >
              <PanelLeftOpen className="size-4" />
            </button>
          )}
          <NotificationBell />
        </div>
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
