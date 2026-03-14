import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useNotifications, useNotificationMutations, useUnreadCount } from "@/hooks/use-notifications"
import { Bell, Check, CheckCheck } from "lucide-react"

const severityColors: Record<string, string> = {
  info: "bg-blue-500",
  warning: "bg-yellow-500",
  critical: "bg-red-500",
}

export function NotificationsPage() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<"all" | "unread">("all")
  const [limit] = useState(50)
  const unreadOnly = filter === "unread"

  const { notifications, total, unreadCount, loading, reload } = useNotifications(limit)
  const { markRead, markAllRead } = useNotificationMutations()
  const { reset, decrement } = useUnreadCount()

  const filtered = unreadOnly
    ? notifications.filter((n) => !n.read)
    : notifications

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

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Bell className="size-5 text-muted-foreground" />
          <h1 className="text-xl font-semibold">{t('notifications.title')}</h1>
          {unreadCount > 0 && (
            <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-500">
              {unreadCount} {t('notifications.unread').toLowerCase()}
            </span>
          )}
        </div>
        {unreadCount > 0 && (
          <button
            onClick={handleMarkAllRead}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          >
            <CheckCheck className="size-3.5" />
            {t('notifications.markAllRead')}
          </button>
        )}
      </div>

      {/* Filter tabs */}
      <div className="flex gap-1 rounded-md bg-muted p-1">
        <button
          onClick={() => setFilter("all")}
          className={`rounded px-3 py-1 text-sm transition-colors ${
            filter === "all"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          {t('notifications.all')} ({total})
        </button>
        <button
          onClick={() => setFilter("unread")}
          className={`rounded px-3 py-1 text-sm transition-colors ${
            filter === "unread"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          {t('notifications.unread')} ({unreadCount})
        </button>
      </div>

      {/* Notification list */}
      {loading ? (
        <div className="py-12 text-center text-sm text-muted-foreground">
          {t('common.loading')}
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-12 text-center text-sm text-muted-foreground">
          {t('notifications.empty')}
        </div>
      ) : (
        <div className="space-y-1">
          {filtered.map((n) => (
            <div
              key={n.id}
              className={`flex items-start gap-3 rounded-lg border px-4 py-3 transition-colors ${
                n.read
                  ? "border-border bg-card"
                  : "border-border bg-accent/30"
              }`}
            >
              <div className={`mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full ${severityColors[n.severity] || severityColors.info}`} />
              <div className="min-w-0 flex-1">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <span className={`text-sm ${n.read ? "text-muted-foreground" : "font-medium"}`}>
                      {n.title}
                    </span>
                    <span className="ml-2 text-xs text-muted-foreground/60">
                      {t(`notifications.types.${n.type}`, { defaultValue: n.type })}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-xs text-muted-foreground/60">
                      {formatDate(n.created_at)}
                    </span>
                    {!n.read && (
                      <button
                        onClick={() => handleMarkRead(n.id)}
                        className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                        title={t('notifications.markRead')}
                      >
                        <Check className="size-3.5" />
                      </button>
                    )}
                  </div>
                </div>
                {n.body && (
                  <p className="mt-0.5 text-xs text-muted-foreground">{n.body}</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
