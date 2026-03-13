import { useState, useMemo, useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { useAdminAnalytics } from "@/hooks/use-admin"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts"
import { TrendingUp, TrendingDown } from "lucide-react"

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

const TIME_RANGES = [
  { label: "24h", hours: 24, bucket: 60 },
  { label: "7d", hours: 168, bucket: 360 },
  { label: "30d", hours: 720, bucket: 1440 },
]

export function AnalyticsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [rangeIdx, setRangeIdx] = useState(0)
  const range = TIME_RANGES[rangeIdx]

  const chart3 = useCssColor("--chart-3")
  const chart5 = useCssColor("--chart-5")
  const chart1 = useCssColor("--chart-1")
  const borderColor = useCssColor("--border")
  const mutedColor = useCssColor("--muted-foreground")
  const cardColor = useCssColor("--card")
  const fgColor = useCssColor("--foreground")

  const since = useMemo(
    () => new Date(Date.now() - range.hours * 3600 * 1000).toISOString(),
    [range.hours]
  )

  const { data, loading, error } = useAdminAnalytics(since, range.bucket)

  const stats = data?.stats
  const topAgents = data?.top_agents ?? []
  const timeSeries = data?.time_series ?? []

  const successRate =
    stats && stats.total_calls > 0
      ? ((stats.success_calls / stats.total_calls) * 100).toFixed(1)
      : "0.0"

  const errorRate =
    stats && stats.total_calls > 0
      ? ((stats.error_calls / stats.total_calls) * 100).toFixed(1)
      : "0.0"

  // Compute trend from time series data (compare first half vs second half)
  const trend = useMemo(() => {
    if (timeSeries.length < 2) return null
    const mid = Math.floor(timeSeries.length / 2)
    const firstHalf = timeSeries.slice(0, mid)
    const secondHalf = timeSeries.slice(mid)
    const sumFirst = firstHalf.reduce((s, p) => s + p.total_calls, 0)
    const sumSecond = secondHalf.reduce((s, p) => s + p.total_calls, 0)
    if (sumFirst === 0) return null
    const pct = ((sumSecond - sumFirst) / sumFirst) * 100
    return { direction: pct >= 0 ? "up" : "down", pct: Math.abs(pct).toFixed(1) }
  }, [timeSeries])

  // Format time series for chart
  const chartData = useMemo(() =>
    timeSeries.map((point) => ({
      ...point,
      time: new Date(point.timestamp).toLocaleString(undefined, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }),
    })),
    [timeSeries]
  )

  // Prepare top agents bar chart data
  const barData = useMemo(() =>
    topAgents.map((a) => ({
      name: a.agent_name || a.agent_id.slice(0, 12),
      total_calls: a.total_calls ?? 0,
      agent_id: a.agent_id,
    })),
    [topAgents]
  )

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
          <h1 className="text-2xl font-bold">{t('adminAnalytics.title')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('adminAnalytics.globalAnalytics')}
          </p>
        </div>
        <div className="flex gap-1">
          {TIME_RANGES.map((r, i) => (
            <Button
              key={r.label}
              size="sm"
              variant={rangeIdx === i ? "default" : "outline"}
              onClick={() => setRangeIdx(i)}
            >
              {r.label}
            </Button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('adminAnalytics.loadingAnalytics')}</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <>
          {/* Stat Cards with trend indicators */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.totalCalls')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats?.total_calls?.toLocaleString() ?? 0}
                </p>
                {trend && (
                  <p className={`flex items-center gap-1 text-xs mt-1 ${trend.direction === "up" ? "text-emerald-500" : "text-red-500"}`}>
                    {trend.direction === "up" ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
                    {trend.pct}% {t('common.vsLastPeriod')}
                  </p>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.successRate')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-emerald-500">{successRate}%</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.avgLatency')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats?.avg_duration_ms?.toFixed(0) ?? 0}ms
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.errorRate')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-red-500">{errorRate}%</p>
              </CardContent>
            </Card>
          </div>

          {/* Invocation Timeline AreaChart */}
          {chartData.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('adminAnalytics.invocationTimeline')}</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <AreaChart data={chartData}>
                    <defs>
                      <linearGradient id="successGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chart3} stopOpacity={0.3} />
                        <stop offset="100%" stopColor={chart3} stopOpacity={0} />
                      </linearGradient>
                      <linearGradient id="errorGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chart5} stopOpacity={0.3} />
                        <stop offset="100%" stopColor={chart5} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke={borderColor} />
                    <XAxis
                      dataKey="time"
                      tick={{ fontSize: 11, fill: mutedColor }}
                      tickLine={false}
                      interval={Math.max(0, Math.ceil(chartData.length / 8) - 1)}
                      angle={-30}
                      textAnchor="end"
                      height={50}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: mutedColor }}
                      tickLine={false}
                      width={40}
                    />
                    <Tooltip
                      contentStyle={tooltipStyle}
                      labelStyle={{ color: fgColor }}
                    />
                    <Legend />
                    <Area
                      type="monotone"
                      dataKey="success_calls"
                      name={t('adminAnalytics.success')}
                      stroke={chart3}
                      strokeWidth={2}
                      fill="url(#successGrad)"
                      dot={false}
                      activeDot={{ r: 4, fill: chart3, strokeWidth: 2, stroke: cardColor }}
                    />
                    <Area
                      type="monotone"
                      dataKey="error_calls"
                      name={t('adminAnalytics.errors')}
                      stroke={chart5}
                      strokeWidth={2}
                      fill="url(#errorGrad)"
                      dot={false}
                      activeDot={{ r: 4, fill: chart5, strokeWidth: 2, stroke: cardColor }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          )}

          {/* Top Agents Horizontal BarChart */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">{t('adminAnalytics.topAgents')}</CardTitle>
            </CardHeader>
            <CardContent>
              {barData.length === 0 ? (
                <div className="flex h-48 items-center justify-center">
                  <p className="text-sm text-muted-foreground">{t('adminAnalytics.noData')}</p>
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={Math.max(200, barData.length * 40 + 40)}>
                  <BarChart data={barData} layout="vertical" margin={{ left: 20 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke={borderColor} horizontal={false} />
                    <XAxis
                      type="number"
                      tick={{ fontSize: 11, fill: mutedColor }}
                      tickLine={false}
                    />
                    <YAxis
                      type="category"
                      dataKey="name"
                      tick={{ fontSize: 11, fill: mutedColor }}
                      tickLine={false}
                      width={120}
                    />
                    <Tooltip
                      contentStyle={tooltipStyle}
                      labelStyle={{ color: fgColor }}
                      cursor={{ fill: `${borderColor}33` }}
                    />
                    <Bar
                      dataKey="total_calls"
                      name={t('adminAnalytics.totalCalls')}
                      fill={chart1}
                      radius={[0, 4, 4, 0]}
                      cursor="pointer"
                      onClick={(entry) => {
                        if (entry?.agent_id) navigate(`/admin/agents/${entry.agent_id}`)
                      }}
                    />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
