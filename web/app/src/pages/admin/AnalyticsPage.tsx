import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useAdminAnalytics } from "@/hooks/use-admin"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const TIME_RANGES = [
  { label: "24h", hours: 24, bucket: 60 },
  { label: "7d", hours: 168, bucket: 360 },
  { label: "30d", hours: 720, bucket: 1440 },
]

export function AnalyticsPage() {
  const { t } = useTranslation()
  const [rangeIdx, setRangeIdx] = useState(0)
  const range = TIME_RANGES[rangeIdx]

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
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.totalCalls')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats?.total_calls?.toLocaleString() ?? 0}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm text-muted-foreground">{t('adminAnalytics.successRate')}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-green-500">{successRate}%</p>
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

          {/* Time Series Chart (text-based since no recharts) */}
          {timeSeries.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('adminAnalytics.invocationTimeline')}</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('adminAnalytics.time')}</TableHead>
                      <TableHead>{t('adminAnalytics.total')}</TableHead>
                      <TableHead>{t('adminAnalytics.success')}</TableHead>
                      <TableHead>{t('adminAnalytics.errors')}</TableHead>
                      <TableHead>{t('adminAnalytics.avgDuration')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {timeSeries.map((point, i) => (
                      <TableRow key={i}>
                        <TableCell className="text-xs text-muted-foreground">
                          {new Date(point.timestamp).toLocaleString()}
                        </TableCell>
                        <TableCell>{point.total_calls}</TableCell>
                        <TableCell className="text-green-500">{point.success_calls}</TableCell>
                        <TableCell className="text-red-500">{point.error_calls}</TableCell>
                        <TableCell>{(point.avg_duration_ms ?? 0).toFixed(0)}ms</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* Top Agents */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">{t('adminAnalytics.topAgents')}</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>#</TableHead>
                    <TableHead>{t('adminAnalytics.agent')}</TableHead>
                    <TableHead>{t('adminAnalytics.totalCalls')}</TableHead>
                    <TableHead>{t('adminAnalytics.success')}</TableHead>
                    <TableHead>{t('adminAnalytics.errors')}</TableHead>
                    <TableHead>{t('adminAnalytics.avgDuration')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {topAgents.map((agent, i) => (
                    <TableRow key={agent.agent_id}>
                      <TableCell className="text-muted-foreground">{i + 1}</TableCell>
                      <TableCell>
                        <div>
                          <p className="font-medium">{agent.agent_name}</p>
                          <p className="text-xs text-muted-foreground font-mono truncate max-w-[200px]">
                            {agent.agent_id}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell className="font-bold">
                        {(agent.total_calls ?? 0).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-green-500">
                        {(agent.success_calls ?? 0).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-red-500">
                        {(agent.error_calls ?? 0).toLocaleString()}
                      </TableCell>
                      <TableCell>{(agent.avg_duration_ms ?? 0).toFixed(0)}ms</TableCell>
                    </TableRow>
                  ))}
                  {topAgents.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                        {t('adminAnalytics.noData')}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
