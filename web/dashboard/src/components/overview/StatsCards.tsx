import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Bot, Wifi, Network, HeartPulse } from "lucide-react"
import type { DashboardStats } from "@/api/types"

interface Props {
  stats: DashboardStats
}

export function StatsCards({ stats }: Props) {
  const activeBridges = stats.bridges.filter((b) => b.available).length

  const cards = [
    {
      title: "Registered Agents",
      value: stats.registered_agents,
      icon: Bot,
      description: "Total registered",
    },
    {
      title: "Online Now",
      value: stats.connected_agents,
      icon: Wifi,
      description: "Connected via signaling",
    },
    {
      title: "Active Bridges",
      value: activeBridges,
      icon: Network,
      description: `of ${stats.bridges.length} configured`,
    },
    {
      title: "System Health",
      value: stats.health.status === "ok" ? "OK" : "Degraded",
      icon: HeartPulse,
      description: `${Object.keys(stats.health.components).length} components`,
      variant: stats.health.status === "ok" ? "default" : ("destructive" as const),
    },
  ]

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
            <card.icon className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div
              className={`text-2xl font-bold ${
                card.variant === "destructive" ? "text-destructive" : ""
              }`}
            >
              {card.value}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {card.description}
            </p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
