import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import type { AgentSummary } from "@/api/types"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { timeAgo } from "@/lib/time"

interface Props {
  agents: AgentSummary[]
}

function statusColor(status: string) {
  switch (status) {
    case "online":
      return "bg-emerald-500"
    case "offline":
      return "bg-zinc-500"
    case "degraded":
      return "bg-amber-500"
    default:
      return "bg-zinc-500"
  }
}

export function EventFeed({ agents }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{t('admin.recentAgents')}</CardTitle>
      </CardHeader>
      <CardContent>
        {agents.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('admin.noAgentsRegistered')}</p>
        ) : (
          <div className="space-y-3">
            {agents.map((agent) => (
              <div
                key={agent.id}
                className="flex items-center justify-between rounded-md border border-border p-3 cursor-pointer hover:bg-accent/50 transition-colors"
                onClick={() => navigate(`/agents/${agent.id}`)}
              >
                <div className="flex items-center gap-3">
                  <span
                    className={`size-2 rounded-full ${statusColor(agent.status)}`}
                  />
                  <div>
                    <p className="text-sm font-medium">{agent.name}</p>
                    <div className="flex gap-1 mt-1">
                      {(agent.protocols ?? []).map((p) => (
                        <Badge key={p} variant="secondary" className="text-[10px] px-1.5 py-0">
                          {p.toUpperCase()}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </div>
                <span className="text-xs text-muted-foreground">
                  {timeAgo(agent.last_heartbeat)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
