import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Bell } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useNotifications, useUnreadCount, useNotificationMutations } from "@/hooks/use-notifications"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const severityColors: Record<string, string> = {
  info: "bg-blue-500",
  warning: "bg-yellow-500",
  critical: "bg-red-500",
}

function timeAgo(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return "just now"
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return `${diffHrs}h ago`
  const diffDays = Math.floor(diffHrs / 24)
  return `${diffDays}d ago`
}

export function NotificationBell() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { count, reset, decrement } = useUnreadCount()
  const { notifications, reload } = useNotifications(10)
  const { markRead, markAllRead } = useNotificationMutations()
  const [open, setOpen] = useState(false)

  const handleOpen = (isOpen: boolean) => {
    setOpen(isOpen)
    if (isOpen) {
      reload()
    }
  }

  const handleMarkRead = async (id: string) => {
    await markRead(id)
    decrement(1)
    reload()
  }

  const handleMarkAllRead = async () => {
    await markAllRead()
    reset()
    reload()
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpen}>
      <DropdownMenuTrigger asChild>
        <button className="relative rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors">
          <Bell className="size-4" />
          {count > 0 && (
            <span className="absolute -top-0.5 -right-0.5 flex size-4 items-center justify-center rounded-full bg-red-500 text-[10px] font-medium text-white">
              {count > 99 ? "99+" : count}
            </span>
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <div className="flex items-center justify-between px-3 py-2">
          <span className="text-sm font-medium">{t('notifications.title')}</span>
          {count > 0 && (
            <button
              onClick={handleMarkAllRead}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              {t('notifications.markAllRead')}
            </button>
          )}
        </div>
        <DropdownMenuSeparator />
        {notifications.length === 0 ? (
          <div className="px-3 py-6 text-center text-sm text-muted-foreground">
            {t('notifications.empty')}
          </div>
        ) : (
          <>
            {notifications.map((n) => (
              <DropdownMenuItem
                key={n.id}
                className="flex items-start gap-2.5 px-3 py-2.5 cursor-pointer"
                onClick={() => {
                  if (!n.read) handleMarkRead(n.id)
                }}
              >
                <div className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${severityColors[n.severity] || severityColors.info}`} />
                <div className="min-w-0 flex-1">
                  <div className={`text-sm truncate ${n.read ? "text-muted-foreground" : "font-medium"}`}>
                    {n.title}
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5 truncate">
                    {n.body}
                  </div>
                  <div className="text-xs text-muted-foreground/60 mt-0.5">
                    {timeAgo(n.created_at)}
                  </div>
                </div>
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="justify-center text-xs text-muted-foreground cursor-pointer"
              onClick={() => {
                setOpen(false)
                navigate("/console/notifications")
              }}
            >
              {t('notifications.viewAll')}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
