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
import { Bot, PhoneCall, CheckCircle, Timer } from "lucide-react"

export function ProviderDashboardPage() {
  const { t } = useTranslation()
  const { data, loading, error } = useProviderDashboard()

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
      <div>
        <h1 className="text-2xl font-bold">{t('provider.dashboard')}</h1>
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
          value={data.total_calls.toLocaleString()}
          icon={PhoneCall}
        />
        <AgentStatsCard
          title={t('provider.successRate')}
          value={`${data.success_rate.toFixed(1)}%`}
          icon={CheckCircle}
        />
        <AgentStatsCard
          title={t('provider.avgLatency')}
          value={`${data.avg_latency_ms.toFixed(0)}ms`}
          icon={Timer}
        />
      </div>

      {/* Claim token section */}
      <ClaimTokenSection />

      {/* Agent list */}
      <div>
        <h2 className="text-lg font-semibold mb-3">{t('provider.myAgents')}</h2>
        {data.agents.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
            <Bot className="size-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">{t('provider.noAgentsRegistered')}</p>
            <Link
              to="/console/register"
              className="text-sm text-primary hover:underline mt-1"
            >
              {t('provider.registerFirst')}
            </Link>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('provider.name')}</TableHead>
                <TableHead>{t('provider.status')}</TableHead>
                <TableHead>{t('provider.calls')}</TableHead>
                <TableHead>{t('provider.successRate')}</TableHead>
                <TableHead>{t('provider.avgLatency')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.agents.map((agent) => (
                <TableRow key={agent.id}>
                  <TableCell>
                    <Link
                      to={`/console/agents/${agent.id}`}
                      className="font-medium text-foreground hover:text-primary transition-colors"
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
                  <TableCell className="text-muted-foreground">
                    {agent.total_calls.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {agent.success_rate.toFixed(1)}%
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {agent.avg_latency_ms.toFixed(0)}ms
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}
