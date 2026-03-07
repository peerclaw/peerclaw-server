import { useCallback } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { useAdminAgent, useAdminMutations } from "@/hooks/use-admin"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, loading, error, refetch } = useAdminAgent(id)
  const { verifyAgent, unverifyAgent, deleteAgent } = useAdminMutations()

  const handleVerify = useCallback(async () => {
    if (!id) return
    try {
      await verifyAgent(id)
      refetch()
    } catch (e) {
      alert(e instanceof Error ? e.message : "Failed")
    }
  }, [id, verifyAgent, refetch])

  const handleUnverify = useCallback(async () => {
    if (!id) return
    try {
      await unverifyAgent(id)
      refetch()
    } catch (e) {
      alert(e instanceof Error ? e.message : "Failed")
    }
  }, [id, unverifyAgent, refetch])

  const handleDelete = useCallback(async () => {
    if (!id || !confirm("Are you sure you want to delete this agent?")) return
    try {
      await deleteAgent(id)
      navigate("/admin/agents")
    } catch (e) {
      alert(e instanceof Error ? e.message : "Failed")
    }
  }, [id, deleteAgent, navigate])

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading agent details...</p>
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

  const { agent, owner, reputation_score, reputation_events, review_summary, invocation_stats } =
    data

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">{agent.name}</h1>
          <p className="text-xs text-muted-foreground font-mono mt-1">{agent.id}</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={handleVerify}>
            Verify
          </Button>
          <Button size="sm" variant="outline" onClick={handleUnverify}>
            Unverify
          </Button>
          <Button size="sm" variant="destructive" onClick={handleDelete}>
            Delete
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">Status</CardTitle>
          </CardHeader>
          <CardContent>
            <Badge
              variant={
                agent.status === "online"
                  ? "default"
                  : agent.status === "degraded"
                  ? "secondary"
                  : "outline"
              }
            >
              {agent.status}
            </Badge>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">Reputation</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {reputation_score !== undefined
                ? `${(reputation_score * 100).toFixed(0)}%`
                : "N/A"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">Reviews</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {review_summary
                ? `${review_summary.average_rating.toFixed(1)} (${review_summary.total_reviews})`
                : "None"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">Total Calls</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {invocation_stats?.total_calls?.toLocaleString() ?? 0}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Agent Metadata */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Agent Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Description</span>
              <p>{agent.description || "N/A"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Version</span>
              <p>{agent.version || "N/A"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Protocols</span>
              <div className="flex gap-1 mt-1">
                {agent.protocols?.map((p) => (
                  <Badge key={p} variant="outline" className="text-xs">
                    {p}
                  </Badge>
                ))}
              </div>
            </div>
            <div>
              <span className="text-muted-foreground">Endpoint</span>
              <p className="font-mono text-xs">{agent.endpoint?.url || "N/A"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Registered At</span>
              <p>{new Date(agent.registered_at).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Last Heartbeat</span>
              <p>
                {agent.last_heartbeat
                  ? new Date(agent.last_heartbeat).toLocaleString()
                  : "Never"}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Owner Info */}
      {owner && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Owner</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div>
                <span className="text-muted-foreground">Email</span>
                <p>{owner.email}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Display Name</span>
                <p>{owner.display_name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Role</span>
                <p>
                  <Badge variant="outline">{owner.role}</Badge>
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Invocation Stats */}
      {invocation_stats && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Invocation Statistics (30 days)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-4 gap-4 text-sm">
              <div>
                <span className="text-muted-foreground">Total Calls</span>
                <p className="text-lg font-bold">
                  {invocation_stats.total_calls.toLocaleString()}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Success</span>
                <p className="text-lg font-bold text-green-500">
                  {invocation_stats.success_calls.toLocaleString()}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Errors</span>
                <p className="text-lg font-bold text-red-500">
                  {invocation_stats.error_calls.toLocaleString()}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Avg Duration</span>
                <p className="text-lg font-bold">
                  {invocation_stats.avg_duration_ms.toFixed(0)}ms
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Reputation Events */}
      {reputation_events && reputation_events.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Recent Reputation Events</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Event Type</TableHead>
                  <TableHead>Weight</TableHead>
                  <TableHead>Score After</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {reputation_events.map((event) => (
                  <TableRow key={event.id}>
                    <TableCell>
                      <Badge variant="outline">{event.event_type}</Badge>
                    </TableCell>
                    <TableCell
                      className={event.weight >= 0 ? "text-green-500" : "text-red-500"}
                    >
                      {event.weight > 0 ? "+" : ""}
                      {event.weight.toFixed(2)}
                    </TableCell>
                    <TableCell>{(event.score_after * 100).toFixed(0)}%</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {new Date(event.created_at).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
