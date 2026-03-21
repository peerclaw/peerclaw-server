import { useState, useCallback, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { useAdminAgents, useAdminMutations } from "@/hooks/use-admin"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Download } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { exportToCSV, exportToJSON } from "@/lib/export"
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
import { SortableHeader } from "@/components/ui/sortable-header"
import { TableSkeleton } from "@/components/ui/table-skeleton"

const PAGE_SIZE = 20

export function AgentsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const [protocol, setProtocol] = useState("")
  const [status, setStatus] = useState("")
  const [sort, setSort] = useState("")
  const [page, setPage] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const { data, loading, error, refetch } = useAdminAgents(
    search || undefined,
    protocol || undefined,
    status || undefined,
    sort || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )
  const { deleteAgent } = useAdminMutations()

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteAgent(id)
        setConfirmDelete(null)
        toast.success(t('toast.agentDeleted'))
        refetch()
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.operationFailed'))
      }
    },
    [deleteAgent, refetch]
  )


  const agents = data?.agents ?? []
  const total = data?.total_count ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  // Reset page if current exceeds total after deletion.
  useEffect(() => {
    if (totalPages > 0 && page >= totalPages) setPage(totalPages - 1)
  }, [totalPages, page])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('adminAgents.title')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('adminAgents.agentsRegistered', { count: total })}
          </p>
        </div>
        {agents.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="outline">
                <Download className="size-4 mr-1.5" />
                {t('common.export')}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => exportToCSV(
                agents.map(a => ({
                  id: a.id,
                  name: a.name,
                  status: a.status,
                  protocols: a.protocols?.join("; ") ?? "",
                  registered_at: a.registered_at ?? "",
                  last_heartbeat: a.last_heartbeat ?? "",
                })),
                [
                  { key: "id", label: "ID" },
                  { key: "name", label: "Name" },
                  { key: "status", label: "Status" },
                  { key: "protocols", label: "Protocols" },
                  { key: "registered_at", label: "Registered At" },
                  { key: "last_heartbeat", label: "Last Heartbeat" },
                ],
                "agents"
              )}>
                {t('common.exportCSV')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => exportToJSON(agents, "agents")}>
                {t('common.exportJSON')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <div className="flex gap-3">
        <Input
          placeholder={t('adminAgents.searchPlaceholder')}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(0)
          }}
          className="max-w-sm"
        />
        <select
          value={protocol}
          onChange={(e) => {
            setProtocol(e.target.value)
            setPage(0)
          }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t('adminAgents.allProtocols')}</option>
          <option value="a2a">A2A</option>
          <option value="mcp">MCP</option>
          <option value="acp">ACP</option>
        </select>
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value)
            setPage(0)
          }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t('adminAgents.allStatus')}</option>
          <option value="online">{t('adminAgents.online')}</option>
          <option value="offline">{t('adminAgents.offline')}</option>
          <option value="degraded">{t('adminAgents.degraded')}</option>
        </select>
      </div>

      {loading ? (
        <Card>
          <CardContent className="p-0">
            <div aria-live="polite"><TableSkeleton rows={5} cols={6} /></div>
          </CardContent>
        </Card>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive" role="alert">{error}</p>
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <SortableHeader field="name" label={t('adminAgents.name')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <TableHead>{t('adminAgents.status')}</TableHead>
                    <TableHead>{t('adminAgents.protocols')}</TableHead>
                    <TableHead>{t('adminAgents.sdkVersion')}</TableHead>
                    <SortableHeader field="registered_at" label={t('adminAgents.lastHeartbeat')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <TableHead className="text-right">{t('adminAgents.actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {agents.map((agent) => (
                    <TableRow
                      key={agent.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/admin/agents/${agent.id}`)}
                    >
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
                      <TableCell className="text-xs text-muted-foreground font-mono">
                        {agent.metadata?.sdk_version || "-"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {agent.last_heartbeat
                          ? new Date(agent.last_heartbeat).toLocaleString()
                          : "Never"}
                      </TableCell>
                      <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                        {confirmDelete === agent.id ? (
                          <span className="inline-flex gap-1">
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleDelete(agent.id)}
                            >
                              {t('common.confirm')}
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => setConfirmDelete(null)}
                            >
                              {t('common.cancel')}
                            </Button>
                          </span>
                        ) : (
                          <Button
                            size="sm"
                            variant="ghost"
                            className="text-destructive"
                            onClick={() => setConfirmDelete(agent.id)}
                          >
                            {t('common.delete')}
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {agents.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                        {t('adminAgents.noAgents')}
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
                {t('common.page')} {page + 1} / {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t('common.previous')}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('common.next')}
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
