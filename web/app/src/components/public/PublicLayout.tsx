import { Outlet, Link, NavLink, useNavigate } from "react-router-dom"
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
import { Github, User, Shield, LogOut, ChevronDown } from "lucide-react"

export function PublicLayout() {
  const { user, logout } = useAuth()
  const { t } = useTranslation()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-50 border-b border-border/60 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2.5 group">
            <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-lg object-cover ring-1 ring-border/50 transition-all group-hover:ring-primary/40" />
            <span className="font-semibold text-sm tracking-tight">PeerClaw</span>
          </Link>

          <nav className="flex items-center gap-1">
            <NavLink
              to="/directory"
              className={({ isActive }) =>
                `rounded-md px-3 py-1.5 text-sm transition-all ${
                  isActive
                    ? "text-foreground font-medium bg-secondary/60"
                    : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
                }`
              }
            >
              {t('nav.directory')}
            </NavLink>
            <NavLink
              to="/playground"
              className={({ isActive }) =>
                `rounded-md px-3 py-1.5 text-sm transition-all ${
                  isActive
                    ? "text-foreground font-medium bg-secondary/60"
                    : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
                }`
              }
            >
              {t('nav.playground')}
            </NavLink>
            <NavLink
              to="/about"
              className={({ isActive }) =>
                `rounded-md px-3 py-1.5 text-sm transition-all ${
                  isActive
                    ? "text-foreground font-medium bg-secondary/60"
                    : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
                }`
              }
            >
              {t('nav.about')}
            </NavLink>
            {user ? (
              <>
                <NavLink
                  to="/console"
                  className={({ isActive }) =>
                    `rounded-md px-3 py-1.5 text-sm transition-all ${
                      isActive
                        ? "text-foreground font-medium bg-secondary/60"
                        : "text-muted-foreground hover:text-foreground hover:bg-secondary/40"
                    }`
                  }
                >
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
            >
              <Github className="size-[18px]" />
            </a>
            <LanguageSwitcher />
          </nav>
        </div>
      </header>

      <main>
        <Outlet />
      </main>
    </div>
  )
}
