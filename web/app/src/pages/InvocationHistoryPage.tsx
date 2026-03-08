import { useState } from "react"
import { useProviderInvocations } from "@/hooks/use-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ChevronLeft, ChevronRight, RefreshCw } from "lucide-react"

const PAGE_SIZE = 20

export function InvocationHistoryPage() {
  const [page, setPage] = useState(1)
  const { data, loading, error, refetch } = useProviderInvocations(page, PAGE_SIZE)

  const statusLabel = (code: number): string => {
    if (code >= 200 && code < 300) return "success"
    if (code === 408) return "timeout"
    if (code >= 400) return "error"
    return String(code)
  }

  const statusVariant = (code: number) => {
    if (code >= 200 && code < 300) return "default" as const
    if (code === 408) return "secondary" as const
    if (code >= 400) return "destructive" as const
    return "outline" as const
  }

  const formatDuration = (ms: number): string => {
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(2)}s`
  }

  const formatTime = (iso: string): string => {
    const date = new Date(iso)
    return date.toLocaleString()
  }

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Invocation History</h1>
          <p className="text-sm text-muted-foreground mt-1">
            View all invocations across your agents
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={refetch} disabled={loading}>
          <RefreshCw className={`size-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      {loading && !data && (
        <div className="flex h-64 items-center justify-center">
          <p className="text-sm text-muted-foreground">Loading invocations...</p>
        </div>
      )}

      {error && (
        <div className="flex h-64 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {data && (
        <>
          {data.invocations.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
              <p className="text-sm text-muted-foreground">No invocations yet.</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Agent</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Time</TableHead>
                  <TableHead>Error</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.invocations.map((inv) => (
                  <TableRow key={inv.id}>
                    <TableCell>
                      <span className="font-medium font-mono text-xs">{inv.agent_id}</span>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(inv.status_code)}>
                        {statusLabel(inv.status_code)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDuration(inv.duration_ms)}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {formatTime(inv.created_at)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-xs truncate">
                      {inv.error || "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-2">
              <p className="text-sm text-muted-foreground">
                Page {page} of {totalPages} ({data.total} total)
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                >
                  <ChevronLeft className="size-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                >
                  Next
                  <ChevronRight className="size-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
