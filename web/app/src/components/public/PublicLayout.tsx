import { useState, useEffect } from "react"
import { Outlet, Link, NavLink, useNavigate, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { Github, User, Shield, LogOut, ChevronDown, Menu } from "lucide-react"
import { Footer } from "./Footer"

export function PublicLayout() {
  const { user, logout } = useAuth()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)

  // Close on route change
  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `rounded-md px-3 py-1.5 text-sm transition-all ${
      isActive
        ? "text-foreground font-medium bg-secondary/60"
        : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
    }`

  const mobileNavLinkClass = ({ isActive }: { isActive: boolean }) =>
    `block rounded-md px-3 py-2.5 text-sm transition-all ${
      isActive
        ? "text-foreground font-medium bg-secondary/60"
        : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
    }`

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:p-4 focus:bg-background focus:text-foreground">
        {t('common.skipToContent')}
      </a>
      <header className="sticky top-0 z-50 border-b border-border/60 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2.5 group">
            <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-lg object-cover ring-1 ring-border/50 transition-all group-hover:ring-primary/40" />
            <span className="font-semibold text-sm tracking-tight">PeerClaw</span>
          </Link>

          {/* Desktop nav */}
          <nav className="hidden md:flex items-center gap-1">
            <NavLink to="/directory" className={navLinkClass}>
              {t('nav.directory')}
            </NavLink>
            <NavLink to="/playground" className={navLinkClass}>
              {t('nav.playground')}
            </NavLink>
            <NavLink to="/about" className={navLinkClass}>
              {t('nav.about')}
            </NavLink>
            {user ? (
              <>
                <NavLink to="/console" className={navLinkClass}>
                  {t('nav.console')}
                </NavLink>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="ml-1 flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm transition-all hover:bg-secondary/40 focus:outline-none">
                      <div className="flex size-6 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary ring-1 ring-primary/20">
                        {(user.display_name || user.email).charAt(0).toUpperCase()}
                      </div>
                      <span className="max-w-[120px] truncate text-sm text-foreground">
                        {user.display_name || user.email}
                      </span>
                      <ChevronDown className="size-3.5 text-muted-foreground" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent side="bottom" align="end" className="w-56">
                    <DropdownMenuLabel className="font-normal">
                      <p className="text-sm font-medium truncate">{user.display_name || user.email}</p>
                      <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => navigate("/console/profile")}>
                      <User />
                      {t("nav.profile")}
                    </DropdownMenuItem>
                    {user.role === "admin" && (
                      <DropdownMenuItem onClick={() => navigate("/admin")}>
                        <Shield />
                        {t("nav.adminPanel")}
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleLogout}>
                      <LogOut />
                      {t("nav.signOut")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </>
            ) : (
              <Link
                to="/login"
                className="ml-1 rounded-lg border border-primary/30 bg-primary/5 px-3.5 py-1.5 text-xs font-medium text-primary transition-all hover:bg-primary/10 hover:border-primary/50"
              >
                {t('nav.signIn')}
              </Link>
            )}
            <a
              href="https://github.com/peerclaw/peerclaw"
              target="_blank"
              rel="noopener noreferrer"
              className="ml-1 rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary/40 transition-all"
              title={t('nav.github')}
              aria-label={t('common.github')}
            >
              <Github className="size-[18px]" />
            </a>
            <LanguageSwitcher />
          </nav>

          {/* Mobile hamburger */}
          <button
            className="md:hidden rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-secondary/40 transition-all"
            onClick={() => setMobileOpen(true)}
            aria-label={t('nav.menu')}
          >
            <Menu className="size-5" />
          </button>

          {/* Mobile sheet */}
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetContent side="right" className="w-64 p-0">
              <SheetTitle className="sr-only">{t('nav.menu')}</SheetTitle>
              <div className="flex flex-col h-full">
                <div className="flex items-center gap-2.5 px-4 py-4 border-b border-border">
                  <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-lg object-cover" />
                  <span className="font-semibold text-sm">PeerClaw</span>
                </div>
                <nav className="flex-1 p-3 space-y-1">
                  <NavLink to="/directory" className={mobileNavLinkClass}>
                    {t('nav.directory')}
                  </NavLink>
                  <NavLink to="/playground" className={mobileNavLinkClass}>
                    {t('nav.playground')}
                  </NavLink>
                  <NavLink to="/about" className={mobileNavLinkClass}>
                    {t('nav.about')}
                  </NavLink>
                  {user && (
                    <>
                      <NavLink to="/console" className={mobileNavLinkClass}>
                        {t('nav.console')}
                      </NavLink>
                      <NavLink to="/console/profile" className={mobileNavLinkClass}>
                        {t('nav.profile')}
                      </NavLink>
                      {user.role === "admin" && (
                        <NavLink to="/admin" className={mobileNavLinkClass}>
                          {t('nav.adminPanel')}
                        </NavLink>
                      )}
                    </>
                  )}
                </nav>
                <div className="border-t border-border p-3 space-y-3">
                  <div className="flex items-center justify-between px-3">
                    <a
                      href="https://github.com/peerclaw/peerclaw"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <Github className="size-4" />
                    </a>
                    <LanguageSwitcher />
                  </div>
                  {user ? (
                    <button
                      onClick={handleLogout}
                      className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground hover:text-foreground hover:bg-secondary/40 transition-colors"
                    >
                      <LogOut className="size-4" />
                      {t('nav.signOut')}
                    </button>
                  ) : (
                    <Link
                      to="/login"
                      className="block rounded-lg border border-primary/30 bg-primary/5 px-3.5 py-2 text-center text-xs font-medium text-primary transition-all hover:bg-primary/10"
                    >
                      {t('nav.signIn')}
                    </Link>
                  )}
                </div>
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </header>

      <main id="main-content" className="flex-1">
        <Outlet />
      </main>

      <Footer />
    </div>
  )
}
