import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderDashboard } from "@/hooks/use-provider"
import { AgentStatsCard } from "@/components/provider/AgentStatsCard"
import { ClaimTokenSection } from "@/components/provider/ClaimTokenSection"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Bot, PhoneCall, CheckCircle, Timer, ArrowRight } from "lucide-react"

export function ProviderDashboardPage() {
  const { t } = useTranslation()
  const { data, loading, error } = useProviderDashboard()

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="size-6 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
          <p className="text-sm text-muted-foreground">{t('provider.loadingDashboard')}</p>
        </div>
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

  const statusColor = (status: string) => {
    switch (status) {
      case "online":
        return "default"
      case "degraded":
        return "secondary"
      default:
        return "outline"
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t('provider.dashboard')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('provider.overview')}
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <AgentStatsCard
          title={t('provider.totalAgents')}
          value={data.total_agents}
          icon={Bot}
        />
        <AgentStatsCard
          title={t('provider.totalCalls')}
          value={(data.total_calls ?? 0).toLocaleString()}
          icon={PhoneCall}
        />
        <AgentStatsCard
          title={t('provider.successRate')}
          value={`${(data.success_rate ?? 0).toFixed(1)}%`}
          icon={CheckCircle}
        />
        <AgentStatsCard
          title={t('provider.avgLatency')}
          value={`${(data.avg_latency_ms ?? 0).toFixed(0)}ms`}
          icon={Timer}
        />
      </div>

      {/* Claim token section */}
      <ClaimTokenSection />

      {/* Agent list */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">{t('provider.myAgents')}</h2>
          {(data.agents ?? []).length > 0 && (
            <Link
              to="/console/agents"
              className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
            >
              View all
              <ArrowRight className="size-3" />
            </Link>
          )}
        </div>

        {(data.agents ?? []).length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 rounded-xl border border-dashed border-border/60">
            <div className="flex size-14 items-center justify-center rounded-2xl bg-primary/8 mb-3">
              <Bot className="size-7 text-primary/50" />
            </div>
            <p className="text-sm text-muted-foreground">{t('provider.noAgentsRegistered')}</p>
            <Link
              to="/console/register"
              className="mt-2 inline-flex items-center gap-1 text-sm text-primary font-medium hover:underline"
            >
              {t('provider.registerFirst')}
              <ArrowRight className="size-3.5" />
            </Link>
          </div>
        ) : (
          <div className="rounded-xl border border-border/60 overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-medium">{t('provider.name')}</TableHead>
                  <TableHead className="text-xs font-medium">{t('provider.status')}</TableHead>
                  <TableHead className="text-xs font-medium">{t('provider.calls')}</TableHead>
                  <TableHead className="text-xs font-medium">{t('provider.successRate')}</TableHead>
                  <TableHead className="text-xs font-medium">{t('provider.avgLatency')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data.agents ?? []).map((agent) => (
                  <TableRow key={agent.id} className="group">
                    <TableCell>
                      <Link
                        to={`/console/agents/${agent.id}`}
                        className="font-medium text-foreground transition-colors group-hover:text-primary"
                      >
                        {agent.name}
                      </Link>
                      <p className="text-xs text-muted-foreground mt-0.5 max-w-xs truncate">
                        {agent.description}
                      </p>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusColor(agent.status)}>
                        {agent.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground tabular-nums font-mono text-xs">
                      {(agent.total_calls ?? 0).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-muted-foreground tabular-nums font-mono text-xs">
                      {(agent.success_rate ?? 0).toFixed(1)}%
                    </TableCell>
                    <TableCell className="text-muted-foreground tabular-nums font-mono text-xs">
                      {(agent.avg_latency_ms ?? 0).toFixed(0)}ms
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  )
}
