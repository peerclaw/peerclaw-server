import { Link } from "react-router-dom"
import type { PublicAgentProfile } from "@/api/types"
import { VerifiedBadge } from "./VerifiedBadge"
import { ReputationMeter } from "./ReputationMeter"

const statusColors: Record<string, string> = {
  online: "bg-emerald-500 shadow-[0_0_6px_oklch(0.72_0.2_160_/_0.5)]",
  offline: "bg-zinc-500",
  degraded: "bg-amber-500 shadow-[0_0_6px_oklch(0.8_0.15_85_/_0.5)]",
}

export function AgentDirectoryCard({ agent }: { agent: PublicAgentProfile }) {
  return (
    <Link
      to={`/agents/${agent.id}`}
      className="group relative block rounded-xl border border-border/60 bg-card p-4 transition-all duration-300 hover:border-primary/30 hover:shadow-[0_0_20px_oklch(0.72_0.15_192_/_0.06)]"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-semibold text-sm text-foreground transition-colors group-hover:text-primary">
              {agent.name}
            </h3>
            <span
              className={`size-2 shrink-0 rounded-full transition-all ${statusColors[agent.status] ?? "bg-zinc-500"}`}
              title={agent.status}
            />
            {agent.verified && <VerifiedBadge />}
          </div>
          {agent.description && (
            <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
              {agent.description}
            </p>
          )}
        </div>
        <div className="shrink-0">
          <ReputationMeter score={agent.reputation_score} size="sm" />
        </div>
      </div>

      {/* Capabilities */}
      <div className="mt-3 flex flex-wrap gap-1.5">
        {agent.capabilities?.slice(0, 3).map((cap) => (
          <span
            key={cap}
            className="rounded-md bg-secondary/80 px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground"
          >
            {cap}
          </span>
        ))}
        {(agent.capabilities?.length ?? 0) > 3 && (
          <span className="text-[10px] text-muted-foreground">
            +{agent.capabilities!.length - 3}
          </span>
        )}
      </div>

      {/* Protocols */}
      {agent.protocols && agent.protocols.length > 0 && (
        <div className="mt-2 flex gap-1.5">
          {agent.protocols.map((p) => (
            <span
              key={p}
              className="rounded-md bg-primary/8 px-1.5 py-0.5 text-[10px] font-mono font-medium text-primary"
            >
              {p}
            </span>
          ))}
        </div>
      )}

      {/* Subtle bottom accent on hover */}
      <div className="absolute bottom-0 left-4 right-4 h-px bg-gradient-to-r from-transparent via-primary/0 to-transparent transition-all duration-300 group-hover:via-primary/40" />
    </Link>
  )
}
