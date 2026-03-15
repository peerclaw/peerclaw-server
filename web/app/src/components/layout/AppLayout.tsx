import { useState, useEffect } from "react"
import { Outlet, useLocation } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useIsMobile } from "@/hooks/use-mobile"
import { Sidebar } from "./Sidebar"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { Menu, PanelLeftClose, PanelLeftOpen } from "lucide-react"

const COLLAPSED_KEY = "peerclaw_admin_sidebar_collapsed"

export function AppLayout() {
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  const location = useLocation()
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

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Desktop sidebar */}
      {!isMobile && (
        <div className="hidden md:flex">
          <Sidebar collapsed={collapsed} />
        </div>
      )}

      {/* Mobile sheet */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-56 p-0">
          <SheetTitle className="sr-only">{t('nav.menu')}</SheetTitle>
          <Sidebar />
        </SheetContent>
      </Sheet>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        {/* Mobile header */}
        {isMobile && (
          <div className="flex h-12 items-center border-b border-border px-4">
            <button
              onClick={() => setMobileOpen(true)}
              className="mr-2 rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
              aria-label={t('nav.menu')}
            >
              <Menu className="size-5" />
            </button>
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <img src="/logo.jpg" alt="PeerClaw" className="size-6 rounded-md object-cover" />
              <span className="font-semibold text-sm truncate">{t('nav.peerclawAdmin')}</span>
            </div>
          </div>
        )}
        {/* Desktop collapse toggle */}
        {!isMobile && (
          <div className="flex h-12 items-center border-b border-border px-6">
            <button
              onClick={toggleCollapsed}
              className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
              title={collapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')}
            >
              {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
            </button>
          </div>
        )}
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
