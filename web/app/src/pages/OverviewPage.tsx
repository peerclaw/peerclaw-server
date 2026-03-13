import { useTranslation } from "react-i18next"
import { useAdminDashboard, useAdminAgents } from "@/hooks/use-admin"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from "recharts"
import { CircleCheck, CircleAlert } from "lucide-react"
import { useEffect, useRef, useState, useMemo } from "react"

function useCssColor(varName: string): string {
  const [color, setColor] = useState("")
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current ?? document.documentElement
    const resolve = () => {
      const val = getComputedStyle(el).getPropertyValue(varName).trim()
      setColor(val || "#888")
    }
    resolve()
    const observer = new MutationObserver(resolve)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })
    return () => observer.disconnect()
  }, [varName])

  return color
}

export function OverviewPage() {
  const { t } = useTranslation()
  const { data, loading, error } = useAdminDashboard()
  // Fetch agents for protocol distribution (first page only, small page size)
  const { data: agentsData } = useAdminAgents(undefined, undefined, undefined, 200, 0)

  const chart1 = useCssColor("--chart-1")
  const chart2 = useCssColor("--chart-2")
  const chart3 = useCssColor("--chart-3")
  const chart4 = useCssColor("--chart-4")
  const chart5 = useCssColor("--chart-5")
  const borderColor = useCssColor("--border")
  const cardColor = useCssColor("--card")
  const fgColor = useCssColor("--foreground")

  const pieColors = [chart1, chart2, chart3, chart4, chart5]

  // Protocol distribution from agents data
  const protocolData = useMemo(() => {
    const agents = agentsData?.agents ?? []
    const counts: Record<string, number> = {}
    for (const agent of agents) {
      for (const proto of agent.protocols ?? []) {
        counts[proto.toUpperCase()] = (counts[proto.toUpperCase()] || 0) + 1
      }
    }
    return Object.entries(counts).map(([name, value]) => ({ name, value }))
  }, [agentsData])

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

  const invocations7d = data.trends?.invocations_7d

  const stats = [
    { label: t('admin.totalUsers'), value: data.total_users ?? 0 },
    { label: t('admin.totalAgents'), value: data.total_agents ?? 0 },
    { label: t('admin.connectedAgents'), value: data.connected_agents ?? 0 },
    { label: t('admin.totalInvocations'), value: data.total_invocations ?? 0, trend: invocations7d },
    { label: t('admin.totalReviews'), value: data.total_reviews ?? 0 },
    { label: t('admin.pendingReports'), value: data.pending_reports ?? 0 },
  ]

  const tooltipStyle = {
    backgroundColor: cardColor,
    border: `1px solid ${borderColor}`,
    borderRadius: 8,
    fontSize: 12,
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('admin.dashboard')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('admin.systemOverview')}
          </p>
        </div>
        {data.health?.status === "ok" ? (
          <span className="inline-flex items-center gap-1.5 text-sm text-emerald-500">
            <CircleCheck className="size-4" />
            {t('admin.healthy')}
          </span>
        ) : (
          <span className="inline-flex items-center gap-1.5 text-sm text-destructive">
            <CircleAlert className="size-4" />
            {t('admin.unhealthy')}
          </span>
        )}
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map(({ label, value, trend }) => (
          <Card key={label}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">{value.toLocaleString()}</p>
              {trend != null && trend > 0 && (
                <p className="text-xs text-muted-foreground mt-1">
                  +{trend.toLocaleString()} this week
                </p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {/* System Health */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">{t('admin.systemHealth')}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{t('admin.database')}</span>
                <span className={`inline-flex items-center gap-1.5 text-sm ${data.health?.database === "ok" ? "text-emerald-500" : "text-destructive"}`}>
                  <span className={`size-2 rounded-full ${data.health?.database === "ok" ? "bg-emerald-500" : "bg-destructive"}`} />
                  {data.health?.database === "ok" ? t('admin.healthy') : (data.health?.database ?? "n/a")}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{t('admin.overallStatus')}</span>
                <span className={`inline-flex items-center gap-1.5 text-sm ${data.health?.status === "ok" ? "text-emerald-500" : "text-destructive"}`}>
                  <span className={`size-2 rounded-full ${data.health?.status === "ok" ? "bg-emerald-500" : "bg-destructive"}`} />
                  {data.health?.status === "ok" ? t('admin.healthy') : (data.health?.status ?? "unknown")}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Protocol Distribution */}
        {protocolData.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">{t('admin.protocolDistribution')}</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={protocolData}
                    cx="50%"
                    cy="50%"
                    innerRadius={45}
                    outerRadius={75}
                    paddingAngle={3}
                    dataKey="value"
                  >
                    {protocolData.map((_, i) => (
                      <Cell key={i} fill={pieColors[i % pieColors.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={tooltipStyle}
                    labelStyle={{ color: fgColor }}
                    formatter={(value: number, name: string) => [`${value} agents`, name]}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
