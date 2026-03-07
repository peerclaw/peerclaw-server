import { Link } from "react-router-dom"
import type { PublicAgentProfile } from "@/api/types"
import { VerifiedBadge } from "./VerifiedBadge"
import { ReputationMeter } from "./ReputationMeter"

const statusColors: Record<string, string> = {
  online: "bg-emerald-500",
  offline: "bg-zinc-500",
  degraded: "bg-yellow-500",
}

export function AgentDirectoryCard({ agent }: { agent: PublicAgentProfile }) {
  return (
    <Link
      to={`/agents/${agent.id}`}
      className="group block rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/40 hover:bg-card/80"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-semibold text-sm text-foreground group-hover:text-primary">
              {agent.name}
            </h3>
            <span
              className={`size-2 rounded-full ${statusColors[agent.status] ?? "bg-zinc-500"}`}
              title={agent.status}
            />
            {agent.verified && <VerifiedBadge />}
          </div>
          {agent.description && (
            <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
              {agent.description}
            </p>
          )}
        </div>
        <div className="w-20 shrink-0">
          <ReputationMeter score={agent.reputation_score} size="sm" />
        </div>
      </div>

      <div className="mt-3 flex flex-wrap gap-1.5">
        {agent.capabilities?.slice(0, 4).map((cap) => (
          <span
            key={cap}
            className="rounded-md bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground"
          >
            {cap}
          </span>
        ))}
        {(agent.capabilities?.length ?? 0) > 4 && (
          <span className="text-[10px] text-muted-foreground">
            +{agent.capabilities!.length - 4}
          </span>
        )}
      </div>

      {agent.protocols && agent.protocols.length > 0 && (
        <div className="mt-2 flex gap-1.5">
          {agent.protocols.map((p) => (
            <span
              key={p}
              className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary"
            >
              {p}
            </span>
          ))}
        </div>
      )}
    </Link>
  )
}
