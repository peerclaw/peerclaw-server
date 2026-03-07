import { Link } from "react-router-dom"
import { useProviderDashboard } from "@/hooks/use-provider"
import { AgentStatsCard } from "@/components/provider/AgentStatsCard"
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
  const { data, loading, error } = useProviderDashboard()

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
        <h1 className="text-2xl font-bold">Provider Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Overview of your agents and performance
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <AgentStatsCard
          title="Total Agents"
          value={data.total_agents}
          icon={Bot}
        />
        <AgentStatsCard
          title="Total Calls"
          value={data.total_calls.toLocaleString()}
          icon={PhoneCall}
        />
        <AgentStatsCard
          title="Success Rate"
          value={`${data.success_rate.toFixed(1)}%`}
          icon={CheckCircle}
        />
        <AgentStatsCard
          title="Avg Latency"
          value={`${data.avg_latency_ms.toFixed(0)}ms`}
          icon={Timer}
        />
      </div>

      {/* Agent list */}
      <div>
        <h2 className="text-lg font-semibold mb-3">My Agents</h2>
        {data.agents.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
            <Bot className="size-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">No agents published yet.</p>
            <Link
              to="/console/publish"
              className="text-sm text-primary hover:underline mt-1"
            >
              Publish your first agent
            </Link>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Calls</TableHead>
                <TableHead>Success Rate</TableHead>
                <TableHead>Avg Latency</TableHead>
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
