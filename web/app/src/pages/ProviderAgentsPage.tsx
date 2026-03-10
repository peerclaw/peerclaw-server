import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderAgents } from "@/hooks/use-provider"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Bot, PlusCircle } from "lucide-react"

export function ProviderAgentsPage() {
  const { t } = useTranslation()
  const { data, loading, error } = useProviderAgents()

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

  const agents = data?.agents ?? []

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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('nav.myAgents')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('directory.agentsRegistered', { count: agents.length })}
          </p>
        </div>
        <Link
          to="/console/publish"
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          <PlusCircle className="size-4" />
          {t('provider.publishAgent')}
        </Link>
      </div>

      {agents.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
          <Bot className="size-8 text-muted-foreground mb-2" />
          <p className="text-sm text-muted-foreground">{t('provider.noAgentsPublished')}</p>
          <Link
            to="/console/publish"
            className="text-sm text-primary hover:underline mt-1"
          >
            {t('provider.publishFirst')}
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
            {agents.map((agent) => (
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
  )
}
