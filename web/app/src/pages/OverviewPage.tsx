import { useStats } from "@/hooks/use-stats"
import { StatsCards } from "@/components/overview/StatsCards"
import { ProtocolChart } from "@/components/overview/ProtocolChart"
import { EventFeed } from "@/components/overview/EventFeed"

export function OverviewPage() {
  const { stats, error, loading } = useStats()

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

  if (!stats) return null

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Overview</h1>
        <p className="text-sm text-muted-foreground mt-1">
          PeerClaw gateway status at a glance
        </p>
      </div>

      <StatsCards stats={stats} />

      <div className="grid gap-6 lg:grid-cols-2">
        <ProtocolChart bridges={stats.bridges} />
        <EventFeed agents={stats.recent_agents} />
      </div>
    </div>
  )
}
