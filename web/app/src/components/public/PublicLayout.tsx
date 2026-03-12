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
      <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <img src="/logo.jpg" alt="PeerClaw" className="size-7 rounded-md object-cover" />
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
              {t('nav.directory')}
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
              {t('nav.playground')}
            </NavLink>
            <NavLink
              to="/about"
              className={({ isActive }) =>
                `text-sm transition-colors ${
                  isActive
                    ? "text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground"
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
                    `text-sm transition-colors ${
                      isActive
                        ? "text-foreground font-medium"
                        : "text-muted-foreground hover:text-foreground"
                    }`
                  }
                >
                  {t('nav.console')}
                </NavLink>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent/50 focus:outline-none">
                      <div className="flex size-6 items-center justify-center rounded-full bg-accent text-xs font-medium">
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
                className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                {t('nav.signIn')}
              </Link>
            )}
            <a
              href="https://github.com/peerclaw/peerclaw"
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground transition-colors"
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
