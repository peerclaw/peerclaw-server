import { useAdminDashboard } from "@/hooks/use-admin"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

export function OverviewPage() {
  const { data, loading, error } = useAdminDashboard()

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading dashboard...</p>
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
    { label: "Total Users", value: data.total_users ?? 0 },
    { label: "Total Agents", value: data.total_agents ?? 0 },
    { label: "Connected Agents", value: data.connected_agents ?? 0 },
    { label: "Total Invocations", value: data.total_invocations ?? 0 },
    { label: "Total Reviews", value: data.total_reviews ?? 0 },
    { label: "Pending Reports", value: data.pending_reports ?? 0 },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Admin Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">
            System overview and health status
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
          <CardTitle className="text-sm font-medium">System Health</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Database</span>
              <Badge variant={data.health?.database === "ok" ? "default" : "destructive"}>
                {data.health?.database ?? "n/a"}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Overall Status</span>
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
