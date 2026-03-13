import { useEffect, useState } from "react"
import { useParams, Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { fetchPublicProfile, fetchReputationHistory, fetchAccessRequestStatus } from "@/api/client"
import type { PublicAgentProfile, ReputationEvent } from "@/api/types"
import { useAuth } from "@/hooks/use-auth"
import { VerifiedBadge } from "@/components/public/VerifiedBadge"
import { TrustedBadge } from "@/components/public/TrustedBadge"
import { ReputationMeter } from "@/components/public/ReputationMeter"
import { ReputationChart } from "@/components/public/ReputationChart"
import { ReviewSection } from "@/components/public/ReviewSection"
import { ReportDialog } from "@/components/public/ReportDialog"
import { AccessRequestDialog } from "@/components/public/AccessRequestDialog"
import { ArrowLeft, ExternalLink, Key, Play, Clock, Star } from "lucide-react"

const statusConfig: Record<string, { color: string; glow: string; label: string }> = {
  online: { color: "bg-emerald-500", glow: "shadow-[0_0_8px_oklch(0.72_0.2_160_/_0.5)]", label: "Online" },
  offline: { color: "bg-zinc-500", glow: "", label: "Offline" },
  degraded: { color: "bg-amber-500", glow: "shadow-[0_0_8px_oklch(0.8_0.15_85_/_0.5)]", label: "Degraded" },
}

export function PublicProfilePage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [agent, setAgent] = useState<PublicAgentProfile | null>(null)
  const [events, setEvents] = useState<ReputationEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [accessStatus, setAccessStatus] = useState<string>("none")

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

  useEffect(() => {
    if (!id || !accessToken) return
    fetchAccessRequestStatus(id, accessToken)
      .then((res) => setAccessStatus(res.status))
      .catch(() => {})
  }, [id, accessToken])

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="size-6 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
          <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
        </div>
      </div>
    )
  }

  if (error || !agent) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-destructive">{error || t('profile.agentNotFound')}</p>
        <Link to="/directory" className="mt-2 text-sm text-primary hover:underline">
          {t('profile.backToDirectory')}
        </Link>
      </div>
    )
  }

  const status = statusConfig[agent.status] ?? statusConfig.offline

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <Link
        to="/directory"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-primary"
      >
        <ArrowLeft className="size-4" />
        {t('profile.backToDirectory')}
      </Link>

      {/* ─── Identity Card ─── */}
      <div className="relative overflow-hidden rounded-2xl border border-border/60 bg-card">
        {/* Subtle gradient accent at top */}
        <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-primary/40 to-transparent" />

        <div className="p-6 sm:p-8">
          <div className="flex flex-col sm:flex-row items-start gap-6">
            {/* Left: Agent info */}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-3 flex-wrap">
                <h1 className="text-2xl font-bold tracking-tight">{agent.name}</h1>
                <div className="flex items-center gap-1.5">
                  <span
                    className={`size-2.5 rounded-full ${status.color} ${status.glow}`}
                    title={agent.status}
                  />
                  <span className="text-xs text-muted-foreground">{status.label}</span>
                </div>
              </div>

              {/* Badges */}
              <div className="mt-2 flex flex-wrap gap-2">
                {agent.verified && <VerifiedBadge />}
                {agent.trusted && <TrustedBadge />}
                {agent.version && (
                  <span className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs font-mono text-secondary-foreground">
                    v{agent.version}
                  </span>
                )}
              </div>

              {agent.description && (
                <p className="mt-3 text-sm text-muted-foreground leading-relaxed">
                  {agent.description}
                </p>
              )}

              {/* Categories */}
              {agent.categories && agent.categories.length > 0 && (
                <div className="mt-3 flex flex-wrap gap-1.5">
                  {agent.categories.map((cat) => (
                    <span
                      key={cat}
                      className="rounded-full bg-secondary/80 px-2.5 py-0.5 text-xs text-secondary-foreground"
                    >
                      {cat}
                    </span>
                  ))}
                </div>
              )}

              {/* Review summary inline */}
              {agent.review_summary && agent.review_summary.total_reviews > 0 && (
                <div className="mt-3 flex items-center gap-2 text-sm">
                  <Star className="size-3.5 fill-amber-400 text-amber-400" />
                  <span className="font-semibold text-amber-400">
                    {agent.review_summary.average_rating.toFixed(1)}
                  </span>
                  <span className="text-muted-foreground">
                    ({agent.review_summary.total_reviews} {t('profile.reviews')})
                  </span>
                </div>
              )}
            </div>

            {/* Right: Reputation gauge */}
            <div className="shrink-0 flex flex-col items-center">
              <ReputationMeter score={agent.reputation_score} size="lg" />
              <p className="mt-2 text-xs text-muted-foreground font-medium">Trust Score</p>
            </div>
          </div>

          {/* Action buttons */}
          <div className="mt-6 flex flex-wrap gap-2">
            {agent.playground_enabled || accessStatus === "approved" ? (
              <Link
                to={`/playground/${agent.id}`}
                className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground transition-all hover:bg-primary/90 hover:shadow-[0_0_20px_oklch(0.72_0.15_192_/_0.25)]"
              >
                <Play className="size-4" />
                {t('profile.tryPlayground')}
              </Link>
            ) : accessStatus === "pending" ? (
              <span className="inline-flex items-center gap-1.5 rounded-xl bg-muted px-5 py-2.5 text-sm font-medium text-muted-foreground cursor-not-allowed">
                <Clock className="size-4" />
                {t('accessRequest.accessPending')}
              </span>
            ) : accessToken ? (
              <AccessRequestDialog
                agentId={agent.id}
                onSubmitted={() => setAccessStatus("pending")}
              />
            ) : (
              <Link
                to="/login"
                className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground transition-all hover:bg-primary/90"
              >
                {t('auth.signIn')}
              </Link>
            )}
            <ReportDialog targetType="agent" targetId={agent.id} />
          </div>

          {/* Public Key */}
          {agent.public_key && (
            <div className="mt-5 flex items-center gap-2.5 rounded-xl bg-secondary/50 px-4 py-2.5 border border-border/40">
              <Key className="size-3.5 shrink-0 text-primary/60" />
              <code className="text-xs text-muted-foreground break-all font-mono leading-relaxed">
                {agent.public_key}
              </code>
            </div>
          )}

          {/* Endpoint */}
          {agent.endpoint_url && (
            <div className="mt-2 flex items-center gap-2 text-sm">
              <ExternalLink className="size-3.5 text-muted-foreground" />
              <a
                href={agent.endpoint_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline break-all font-mono text-xs"
              >
                {agent.endpoint_url}
              </a>
            </div>
          )}

          {/* Meta row */}
          <div className="mt-4 flex flex-wrap gap-4 text-xs text-muted-foreground border-t border-border/40 pt-4">
            <span>
              {t('profile.registered')}{" "}
              {new Date(agent.registered_at).toLocaleDateString()}
            </span>
            {agent.verified_at && (
              <span>
                {t('profile.verified')}{" "}
                {new Date(agent.verified_at).toLocaleDateString()}
              </span>
            )}
            <span className="font-mono">{t('profile.id')}: {agent.id}</span>
          </div>
        </div>
      </div>

      {/* ─── Capabilities & Protocols ─── */}
      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {agent.capabilities && agent.capabilities.length > 0 && (
          <div className="rounded-xl border border-border/60 bg-card p-5">
            <h2 className="mb-3 text-sm font-semibold">{t('profile.capabilities')}</h2>
            <div className="flex flex-wrap gap-1.5">
              {agent.capabilities.map((cap) => (
                <span
                  key={cap}
                  className="rounded-md bg-secondary/80 px-2 py-0.5 text-xs text-secondary-foreground"
                >
                  {cap}
                </span>
              ))}
            </div>
          </div>
        )}

        {agent.protocols && agent.protocols.length > 0 && (
          <div className="rounded-xl border border-border/60 bg-card p-5">
            <h2 className="mb-3 text-sm font-semibold">{t('profile.protocols')}</h2>
            <div className="flex flex-wrap gap-1.5">
              {agent.protocols.map((p) => (
                <span
                  key={p}
                  className="rounded-md bg-primary/8 px-2 py-0.5 text-xs font-mono font-medium text-primary"
                >
                  {p}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ─── Skills ─── */}
      {agent.skills && agent.skills.length > 0 && (
        <div className="mt-4 rounded-xl border border-border/60 bg-card p-5">
          <h2 className="mb-3 text-sm font-semibold">{t('profile.skills')}</h2>
          <div className="space-y-2.5">
            {agent.skills.map((skill) => (
              <div key={skill.name} className="rounded-lg bg-secondary/30 px-3 py-2">
                <p className="text-sm font-medium">{skill.name}</p>
                {skill.description && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {skill.description}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ─── Reviews ─── */}
      <div className="mt-4">
        <ReviewSection agentId={agent.id} />
      </div>

      {/* ─── Reputation History ─── */}
      <div className="mt-4 rounded-xl border border-border/60 bg-card p-5">
        <h2 className="mb-4 text-sm font-semibold">{t('profile.reputationHistory')}</h2>
        <ReputationChart events={events} />

        {events.length > 0 && (
          <div className="mt-4 max-h-64 overflow-y-auto rounded-lg">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-border/60 text-left text-muted-foreground">
                  <th className="pb-2 pr-3 font-medium">{t('profile.event')}</th>
                  <th className="pb-2 pr-3 font-medium">{t('profile.weight')}</th>
                  <th className="pb-2 pr-3 font-medium">{t('profile.scoreAfter')}</th>
                  <th className="pb-2 font-medium">{t('profile.time')}</th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id} className="border-b border-border/30 hover:bg-secondary/20 transition-colors">
                    <td className="py-2 pr-3 font-mono text-foreground/80">
                      {e.event_type}
                    </td>
                    <td
                      className={`py-2 pr-3 font-mono ${
                        e.weight >= 0 ? "text-emerald-400" : "text-red-400"
                      }`}
                    >
                      {e.weight > 0 ? "+" : ""}
                      {e.weight.toFixed(1)}
                    </td>
                    <td className="py-2 pr-3 tabular-nums font-mono">
                      {Math.round(e.score_after * 100)}
                    </td>
                    <td className="py-2 text-muted-foreground">
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
