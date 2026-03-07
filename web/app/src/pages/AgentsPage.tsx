import { useState, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useAdminAgents, useAdminMutations } from "@/hooks/use-admin"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Card,
  CardContent,
} from "@/components/ui/card"

export function AgentsPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const [protocol, setProtocol] = useState("")
  const [status, setStatus] = useState("")
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const { data, loading, error, refetch } = useAdminAgents(
    search || undefined,
    protocol || undefined,
    status || undefined
  )
  const { deleteAgent, verifyAgent, unverifyAgent } = useAdminMutations()

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteAgent(id)
        setConfirmDelete(null)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to delete agent")
      }
    },
    [deleteAgent, refetch]
  )

  const handleVerify = useCallback(
    async (id: string) => {
      try {
        await verifyAgent(id)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to verify agent")
      }
    },
    [verifyAgent, refetch]
  )

  const handleUnverify = useCallback(
    async (id: string) => {
      try {
        await unverifyAgent(id)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to unverify agent")
      }
    },
    [unverifyAgent, refetch]
  )

  const agents = data?.agents ?? []
  const total = data?.total_count ?? 0

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Agents</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {total} registered agent{total !== 1 ? "s" : ""}
        </p>
      </div>

      <div className="flex gap-3">
        <Input
          placeholder="Search agents..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-sm"
        />
        <select
          value={protocol}
          onChange={(e) => setProtocol(e.target.value)}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Protocols</option>
          <option value="a2a">A2A</option>
          <option value="mcp">MCP</option>
          <option value="acp">ACP</option>
        </select>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Status</option>
          <option value="online">Online</option>
          <option value="offline">Offline</option>
          <option value="degraded">Degraded</option>
        </select>
      </div>

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">Loading agents...</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Protocols</TableHead>
                  <TableHead>Verified</TableHead>
                  <TableHead>Last Heartbeat</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {agents.map((agent) => (
                  <TableRow key={agent.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{agent.name}</p>
                        <p className="text-xs text-muted-foreground font-mono truncate max-w-[200px]">
                          {agent.id}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell>
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
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {agent.protocols?.map((p) => (
                          <Badge key={p} variant="outline" className="text-xs">
                            {p}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell>
                      {agent.peerclaw?.reputation_score !== undefined && (
                        <span className="text-xs text-muted-foreground">
                          {(agent.peerclaw.reputation_score * 100).toFixed(0)}%
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {agent.last_heartbeat
                        ? new Date(agent.last_heartbeat).toLocaleString()
                        : "Never"}
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      {confirmDelete === agent.id ? (
                        <span className="inline-flex gap-1">
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={() => handleDelete(agent.id)}
                          >
                            Confirm
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => setConfirmDelete(null)}
                          >
                            Cancel
                          </Button>
                        </span>
                      ) : (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => navigate(`/admin/agents/${agent.id}`)}
                          >
                            View
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleVerify(agent.id)}
                          >
                            Verify
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleUnverify(agent.id)}
                          >
                            Unverify
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            className="text-destructive"
                            onClick={() => setConfirmDelete(agent.id)}
                          >
                            Delete
                          </Button>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {agents.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      No agents found
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
