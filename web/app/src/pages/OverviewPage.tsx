import { useTranslation } from "react-i18next"
import { useAdminDashboard } from "@/hooks/use-admin"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

export function OverviewPage() {
  const { t } = useTranslation()
  const { data, loading, error } = useAdminDashboard()

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">{t('provider.loadingDashboard')}</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-destructive">{error}</p>
      </div>
    )
  }

  if (!data) return null

  const stats = [
    { label: t('admin.totalUsers'), value: data.total_users ?? 0 },
    { label: t('admin.totalAgents'), value: data.total_agents ?? 0 },
    { label: t('admin.connectedAgents'), value: data.connected_agents ?? 0 },
    { label: t('admin.totalInvocations'), value: data.total_invocations ?? 0 },
    { label: t('admin.totalReviews'), value: data.total_reviews ?? 0 },
    { label: t('admin.pendingReports'), value: data.pending_reports ?? 0 },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('admin.dashboard')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('admin.systemOverview')}
          </p>
        </div>
        <Badge variant={data.health?.status === "ok" ? "default" : "destructive"}>
          {data.health?.status ?? "unknown"}
        </Badge>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map(({ label, value }) => (
          <Card key={label}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">{value.toLocaleString()}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">{t('admin.systemHealth')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">{t('admin.database')}</span>
              <Badge variant={data.health?.database === "ok" ? "default" : "destructive"}>
                {data.health?.database ?? "n/a"}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">{t('admin.overallStatus')}</span>
              <Badge variant={data.health?.status === "ok" ? "default" : "destructive"}>
                {data.health?.status ?? "unknown"}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
