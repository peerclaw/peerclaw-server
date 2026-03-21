import { useState, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { useAdminAudit } from "@/hooks/use-admin"
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
import { TableSkeleton } from "@/components/ui/table-skeleton"

const PAGE_SIZE = 20

const DATE_RANGES = [
  { label: "Today", value: () => new Date(new Date().setHours(0, 0, 0, 0)).toISOString() },
  { label: "7d", value: () => new Date(Date.now() - 7 * 86400000).toISOString() },
  { label: "30d", value: () => new Date(Date.now() - 30 * 86400000).toISOString() },
  { label: "All", value: () => "" },
]

export function AuditLogPage() {
  const { t } = useTranslation()
  const [adminFilter, setAdminFilter] = useState("")
  const [actionFilter, setActionFilter] = useState("")
  const [targetTypeFilter, setTargetTypeFilter] = useState("")
  const [sinceFilter, setSinceFilter] = useState("")
  const [page, setPage] = useState(0)

  const { data, loading, error } = useAdminAudit(
    adminFilter || undefined,
    actionFilter || undefined,
    targetTypeFilter || undefined,
    sinceFilter || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  useEffect(() => {
    if (totalPages > 0 && page >= totalPages) setPage(totalPages - 1)
  }, [totalPages, page])

  const formatTime = (iso: string) => {
    const d = new Date(iso)
    return d.toLocaleString()
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("adminAudit.title")}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t("adminAudit.description")}</p>
      </div>

      <div className="flex flex-wrap gap-3">
        <Input
          placeholder={t("adminAudit.filterAdmin")}
          value={adminFilter}
          onChange={(e) => { setAdminFilter(e.target.value); setPage(0) }}
          className="max-w-[200px]"
        />
        <select
          value={actionFilter}
          onChange={(e) => { setActionFilter(e.target.value); setPage(0) }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t("adminAudit.filterAction")}</option>
          <option value="user.delete">user.delete</option>
          <option value="user.role_change">user.role_change</option>
          <option value="agent.delete">agent.delete</option>
          <option value="agent.verify">agent.verify</option>
          <option value="agent.unverify">agent.unverify</option>
          <option value="report.update">report.update</option>
          <option value="report.delete">report.delete</option>
          <option value="category.create">category.create</option>
          <option value="category.update">category.update</option>
          <option value="category.delete">category.delete</option>
        </select>
        <select
          value={targetTypeFilter}
          onChange={(e) => { setTargetTypeFilter(e.target.value); setPage(0) }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t("adminAudit.filterTarget")}</option>
          <option value="user">user</option>
          <option value="agent">agent</option>
          <option value="report">report</option>
          <option value="category">category</option>
        </select>
        <div className="flex gap-1">
          {DATE_RANGES.map((range_) => (
            <Button
              key={range_.label}
              size="sm"
              variant={sinceFilter === (range_.value() || "") ? "default" : "outline"}
              onClick={() => { setSinceFilter(range_.value()); setPage(0) }}
            >
              {range_.label}
            </Button>
          ))}
        </div>
      </div>

      {loading ? (
        <Card>
          <CardContent className="p-0">
            <TableSkeleton rows={5} cols={6} />
          </CardContent>
        </Card>
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
                    <TableHead>{t("adminAudit.timestamp")}</TableHead>
                    <TableHead>{t("adminAudit.adminUser")}</TableHead>
                    <TableHead>{t("adminAudit.action")}</TableHead>
                    <TableHead>{t("adminAudit.targetType")}</TableHead>
                    <TableHead>{t("adminAudit.targetId")}</TableHead>
                    <TableHead>{t("adminAudit.details")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(data?.events ?? []).map((event) => (
                    <TableRow key={event.id}>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {formatTime(event.created_at)}
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[120px] truncate">
                        {event.admin_user_id}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{event.action}</Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{event.target_type}</Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[120px] truncate">
                        {event.target_id}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground max-w-[200px] truncate">
                        {event.details || "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(data?.events ?? []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                        {t("adminAudit.noEvents")}
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
                {t("common.page")} {page + 1} / {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t("common.previous")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t("common.next")}
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
