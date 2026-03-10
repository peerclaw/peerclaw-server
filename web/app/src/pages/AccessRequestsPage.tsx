import { useState, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { fetchMyAccessRequests } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"
import { Badge } from "@/components/ui/badge"
import type { AccessRequest } from "@/api/types"

export function AccessRequestsPage() {
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [requests, setRequests] = useState<AccessRequest[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!accessToken) return
    setLoading(true)
    fetchMyAccessRequests(accessToken)
      .then((res) => setRequests(res.requests ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [accessToken])

  const statusVariant = (status: string) => {
    switch (status) {
      case "approved":
        return "default" as const
      case "pending":
        return "secondary" as const
      default:
        return "outline" as const
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t('accessRequest.title')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('accessRequest.myRequestsDesc')}
        </p>
      </div>

      {loading ? (
        <div className="flex h-32 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
        </div>
      ) : requests.length === 0 ? (
        <div className="flex h-32 items-center justify-center rounded-lg border border-border bg-card">
          <p className="text-sm text-muted-foreground">{t('accessRequest.noRequests')}</p>
        </div>
      ) : (
        <div className="rounded-lg border border-border bg-card">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left">
                <th className="p-3 font-medium text-muted-foreground">
                  {t('accessRequest.agentId')}
                </th>
                <th className="p-3 font-medium text-muted-foreground">
                  {t('invocations.status')}
                </th>
                <th className="p-3 font-medium text-muted-foreground">
                  {t('accessRequest.message')}
                </th>
                <th className="p-3 font-medium text-muted-foreground">
                  {t('invocations.time')}
                </th>
              </tr>
            </thead>
            <tbody>
              {requests.map((req) => (
                <tr key={req.id} className="border-b border-border/50">
                  <td className="p-3 font-mono text-xs">{req.agent_id}</td>
                  <td className="p-3">
                    <Badge variant={statusVariant(req.status)}>{req.status}</Badge>
                  </td>
                  <td className="p-3 text-muted-foreground max-w-xs truncate">
                    {req.message || "-"}
                  </td>
                  <td className="p-3 text-xs text-muted-foreground">
                    {new Date(req.created_at).toLocaleDateString()}
                    {req.expires_at && (
                      <span className="ml-2">
                        ({t('accessRequest.expiresAt')}: {new Date(req.expires_at).toLocaleDateString()})
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
