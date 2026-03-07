import { useEffect, useState } from "react"
import { useParams, Link } from "react-router-dom"
import { fetchPublicProfile, fetchReputationHistory } from "@/api/client"
import type { PublicAgentProfile, ReputationEvent } from "@/api/types"
import { VerifiedBadge } from "@/components/public/VerifiedBadge"
import { ReputationMeter } from "@/components/public/ReputationMeter"
import { ReputationChart } from "@/components/public/ReputationChart"
import { ArrowLeft, ExternalLink, Key } from "lucide-react"

const statusColors: Record<string, string> = {
  online: "bg-emerald-500",
  offline: "bg-zinc-500",
  degraded: "bg-yellow-500",
}

export function PublicProfilePage() {
  const { id } = useParams<{ id: string }>()
  const [agent, setAgent] = useState<PublicAgentProfile | null>(null)
  const [events, setEvents] = useState<ReputationEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  useEffect(() => {
    if (!id) return
    setLoading(true)
    Promise.all([
      fetchPublicProfile(id),
      fetchReputationHistory(id, 100),
    ])
      .then(([profile, rep]) => {
        setAgent(profile)
        setEvents(rep.events ?? [])
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground text-sm">
        Loading...
      </div>
    )
  }

  if (error || !agent) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-destructive">{error || "Agent not found"}</p>
        <Link to="/directory" className="mt-2 text-sm text-primary hover:underline">
          Back to directory
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <Link
        to="/directory"
        className="mb-6 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-4" />
        Back to directory
      </Link>

      {/* Identity Header */}
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{agent.name}</h1>
              <span
                className={`size-2.5 rounded-full ${statusColors[agent.status] ?? "bg-zinc-500"}`}
                title={agent.status}
              />
              {agent.verified && <VerifiedBadge />}
            </div>
            {agent.description && (
              <p className="mt-2 text-sm text-muted-foreground">
                {agent.description}
              </p>
            )}
            {agent.version && (
              <p className="mt-1 text-xs text-muted-foreground">
                v{agent.version}
              </p>
            )}
          </div>
          <div className="w-32 shrink-0">
            <ReputationMeter score={agent.reputation_score} size="lg" />
          </div>
        </div>

        {/* Public Key */}
        {agent.public_key && (
          <div className="mt-4 flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2">
            <Key className="size-3.5 text-muted-foreground" />
            <code className="text-xs text-muted-foreground break-all font-mono">
              {agent.public_key}
            </code>
          </div>
        )}

        {/* Endpoint (if public) */}
        {agent.endpoint_url && (
          <div className="mt-2 flex items-center gap-2 text-sm">
            <ExternalLink className="size-3.5 text-muted-foreground" />
            <a
              href={agent.endpoint_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline break-all"
            >
              {agent.endpoint_url}
            </a>
          </div>
        )}

        {/* Meta row */}
        <div className="mt-4 flex flex-wrap gap-4 text-xs text-muted-foreground">
          <span>
            Registered{" "}
            {new Date(agent.registered_at).toLocaleDateString()}
          </span>
          {agent.verified_at && (
            <span>
              Verified{" "}
              {new Date(agent.verified_at).toLocaleDateString()}
            </span>
          )}
          <span>ID: {agent.id}</span>
        </div>
      </div>

      {/* Capabilities & Protocols */}
      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {agent.capabilities && agent.capabilities.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-4">
            <h2 className="mb-2 text-sm font-semibold">Capabilities</h2>
            <div className="flex flex-wrap gap-1.5">
              {agent.capabilities.map((cap) => (
                <span
                  key={cap}
                  className="rounded-md bg-secondary px-2 py-0.5 text-xs text-secondary-foreground"
                >
                  {cap}
                </span>
              ))}
            </div>
          </div>
        )}

        {agent.protocols && agent.protocols.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-4">
            <h2 className="mb-2 text-sm font-semibold">Protocols</h2>
            <div className="flex flex-wrap gap-1.5">
              {agent.protocols.map((p) => (
                <span
                  key={p}
                  className="rounded bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
                >
                  {p}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Skills */}
      {agent.skills && agent.skills.length > 0 && (
        <div className="mt-4 rounded-lg border border-border bg-card p-4">
          <h2 className="mb-2 text-sm font-semibold">Skills</h2>
          <div className="space-y-2">
            {agent.skills.map((skill) => (
              <div key={skill.name}>
                <p className="text-sm font-medium">{skill.name}</p>
                {skill.description && (
                  <p className="text-xs text-muted-foreground">
                    {skill.description}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Reputation History */}
      <div className="mt-4 rounded-lg border border-border bg-card p-4">
        <h2 className="mb-4 text-sm font-semibold">Reputation History</h2>
        <ReputationChart events={events} />

        {events.length > 0 && (
          <div className="mt-4 max-h-64 overflow-y-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-border text-left text-muted-foreground">
                  <th className="pb-2 pr-3">Event</th>
                  <th className="pb-2 pr-3">Weight</th>
                  <th className="pb-2 pr-3">Score After</th>
                  <th className="pb-2">Time</th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id} className="border-b border-border/50">
                    <td className="py-1.5 pr-3 font-mono">
                      {e.event_type}
                    </td>
                    <td
                      className={`py-1.5 pr-3 ${
                        e.weight >= 0 ? "text-emerald-400" : "text-red-400"
                      }`}
                    >
                      {e.weight > 0 ? "+" : ""}
                      {e.weight.toFixed(1)}
                    </td>
                    <td className="py-1.5 pr-3 tabular-nums">
                      {Math.round(e.score_after * 100)}
                    </td>
                    <td className="py-1.5 text-muted-foreground">
                      {new Date(e.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
