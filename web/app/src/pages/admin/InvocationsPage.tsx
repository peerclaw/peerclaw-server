import { useState } from "react"
import { useAdminInvocations } from "@/hooks/use-admin"
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

const PAGE_SIZE = 50

export function InvocationsPage() {
  const [agentFilter, setAgentFilter] = useState("")
  const [userFilter, setUserFilter] = useState("")
  const [page, setPage] = useState(0)

  const { data, loading, error } = useAdminInvocations(
    agentFilter || undefined,
    userFilter || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const invocations = data?.invocations ?? []

  const statusBadgeVariant = (code: number) => {
    if (code >= 200 && code < 300) return "default"
    if (code >= 400 && code < 500) return "secondary"
    if (code >= 500) return "destructive"
    return "outline"
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Invocations</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {total} invocation record{total !== 1 ? "s" : ""}
        </p>
      </div>

      <div className="flex gap-3">
        <Input
          placeholder="Filter by Agent ID..."
          value={agentFilter}
          onChange={(e) => {
            setAgentFilter(e.target.value)
            setPage(0)
          }}
          className="max-w-xs"
        />
        <Input
          placeholder="Filter by User ID..."
          value={userFilter}
          onChange={(e) => {
            setUserFilter(e.target.value)
            setPage(0)
          }}
          className="max-w-xs"
        />
      </div>

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">Loading invocations...</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Agent ID</TableHead>
                    <TableHead>User ID</TableHead>
                    <TableHead>Protocol</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead>Error</TableHead>
                    <TableHead>Created At</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invocations.map((inv) => (
                    <TableRow key={inv.id}>
                      <TableCell className="font-mono text-xs max-w-[80px] truncate">
                        {inv.id.slice(0, 8)}...
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[100px] truncate">
                        {inv.agent_id}
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[100px] truncate">
                        {inv.user_id || "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {inv.protocol || "unknown"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusBadgeVariant(inv.status_code)}>
                          {inv.status_code || "N/A"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs">
                        {inv.duration_ms}ms
                      </TableCell>
                      <TableCell className="text-xs text-red-500 max-w-[150px] truncate">
                        {inv.error || "-"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {new Date(inv.created_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                  {invocations.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                        No invocations found
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Page {page + 1} of {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Previous
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
