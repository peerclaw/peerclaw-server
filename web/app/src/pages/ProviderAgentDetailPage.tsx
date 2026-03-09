import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  useProviderAgent,
  useAgentAnalytics,
  useProviderMutations,
} from "@/hooks/use-provider"
import { AnalyticsChart } from "@/components/provider/AnalyticsChart"
import { AgentStatsCard } from "@/components/provider/AgentStatsCard"
import { ContactsSection } from "@/components/provider/ContactsSection"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  PhoneCall,
  CheckCircle,
  Timer,
  Pencil,
  Trash2,
  ExternalLink,
  ArrowLeft,
} from "lucide-react"

export function ProviderAgentDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: agent, loading, error } = useProviderAgent(id)
  const { data: analytics } = useAgentAnalytics(id)
  const { deleteAgent } = useProviderMutations()
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const handleDelete = async () => {
    if (!id) return
    const confirmed = window.confirm(
      t('provider.deleteConfirm')
    )
    if (!confirmed) return

    setDeleting(true)
    setDeleteError(null)
    try {
      await deleteAgent(id)
      navigate("/console/agents")
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : "Failed to delete agent")
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">{t('provider.loadingAgent')}</p>
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

  if (!agent) return null

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
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <button
            onClick={() => navigate("/console/agents")}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-2 transition-colors"
          >
            <ArrowLeft className="size-3" />
            {t('provider.backToAgents')}
          </button>
          <h1 className="text-2xl font-bold">{agent.name}</h1>
          <p className="text-sm text-muted-foreground mt-1">{agent.description}</p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate(`/console/agents/${id}/edit`)}
          >
            <Pencil className="size-4" />
            {t('common.edit')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleDelete}
            disabled={deleting}
          >
            <Trash2 className="size-4" />
            {deleting ? t('provider.deleting') : t('common.delete')}
          </Button>
        </div>
      </div>

      {deleteError && (
        <p className="text-sm text-destructive">{deleteError}</p>
      )}

      {/* Info card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">{t('provider.agentDetails')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <span className="text-muted-foreground">{t('provider.status')}</span>
              <div className="mt-1">
                <Badge variant={statusColor(agent.status)}>{agent.status}</Badge>
              </div>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.version')}</span>
              <p className="mt-1 font-medium">{agent.version}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.endpoint')}</span>
              <div className="mt-1 flex items-center gap-1">
                <span className="font-mono text-xs truncate max-w-xs">
                  {agent.endpoint_url}
                </span>
                <a
                  href={agent.endpoint_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground"
                >
                  <ExternalLink className="size-3" />
                </a>
              </div>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.authType')}</span>
              <p className="mt-1 font-medium">{agent.auth_type}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.capabilities')}</span>
              <div className="flex flex-wrap gap-1 mt-1">
                {agent.capabilities.map((cap) => (
                  <Badge key={cap} variant="secondary">
                    {cap}
                  </Badge>
                ))}
              </div>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.protocols')}</span>
              <div className="flex flex-wrap gap-1 mt-1">
                {agent.protocols.map((proto) => (
                  <Badge key={proto} variant="outline">
                    {proto.toUpperCase()}
                  </Badge>
                ))}
              </div>
            </div>
            {agent.tags.length > 0 && (
              <div className="sm:col-span-2">
                <span className="text-muted-foreground">{t('provider.tags')}</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {agent.tags.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            <div>
              <span className="text-muted-foreground">{t('provider.created')}</span>
              <p className="mt-1 text-xs">
                {new Date(agent.created_at).toLocaleDateString()}
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t('provider.lastUpdated')}</span>
              <p className="mt-1 text-xs">
                {new Date(agent.updated_at).toLocaleDateString()}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Contacts Whitelist */}
      {id && <ContactsSection agentId={id} />}

      {/* Analytics */}
      {analytics && (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <AgentStatsCard
              title={t('provider.totalCalls')}
              value={analytics.total_calls.toLocaleString()}
              icon={PhoneCall}
            />
            <AgentStatsCard
              title={t('provider.successRate')}
              value={`${analytics.success_rate.toFixed(1)}%`}
              icon={CheckCircle}
            />
            <AgentStatsCard
              title={t('provider.avgLatency')}
              value={`${analytics.avg_latency_ms.toFixed(0)}ms`}
              icon={Timer}
            />
          </div>

          {analytics.time_series.length > 0 && (
            <AnalyticsChart data={analytics.time_series} />
          )}
        </>
      )}
    </div>
  )
}
