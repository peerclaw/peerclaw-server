import { useState, useCallback, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { useAdminReports, useAdminMutations } from "@/hooks/use-admin"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import {
  TableCell,
  TableHead,
} from "@/components/ui/table"
import {
  Card,
  CardContent,
} from "@/components/ui/card"
import { SelectableTable } from "@/components/ui/selectable-table"
import { SortableHeader } from "@/components/ui/sortable-header"
import { TableSkeleton } from "@/components/ui/table-skeleton"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"

import { DEFAULT_PAGE_SIZE } from "@/lib/constants"

const PAGE_SIZE = DEFAULT_PAGE_SIZE

export function ReportsPage() {
  const { t } = useTranslation()
  const [statusFilter, setStatusFilter] = useState("")
  const [search, setSearch] = useState("")
  const [sort, setSort] = useState("")
  const [page, setPage] = useState(0)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

  const STATUS_TABS = [
    { label: t('common.all'), value: "" },
    { label: t('adminReports.pending'), value: "pending" },
    { label: t('adminReports.reviewed'), value: "reviewed" },
    { label: t('adminReports.dismissed'), value: "dismissed" },
    { label: t('adminReports.actioned'), value: "actioned" },
  ]

  const { data, isLoading, error } = useAdminReports(
    statusFilter || undefined,
    search || undefined,
    sort || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )
  const { updateReport, deleteReport, bulkReportsAction } = useAdminMutations()

  const handleStatusChange = useCallback(
    async (id: string, newStatus: string) => {
      try {
        await updateReport.mutateAsync({ id, status: newStatus })
        toast.success(t('toast.reportUpdated'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.operationFailed'))
      }
    },
    [updateReport, t]
  )

  const handleDelete = useCallback(
    async () => {
      if (!deleteTarget) return
      try {
        await deleteReport.mutateAsync(deleteTarget)
        setDeleteTarget(null)
        toast.success(t('toast.reportDeleted'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.operationFailed'))
      }
    },
    [deleteTarget, deleteReport, t]
  )

  const handleBulkAction = useCallback(
    async (action: string, ids: string[]) => {
      try {
        const result = await bulkReportsAction.mutateAsync({ action, ids })
        toast.success(t("common.bulkActions") + `: ${result.success} success, ${result.errors} errors`)
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t("toast.operationFailed"))
      }
    },
    [bulkReportsAction, t]
  )

  const reportBulkActions = [
    { label: t("adminReports.review"), onClick: (ids: string[]) => handleBulkAction("review", ids) },
    { label: t("adminReports.dismiss"), onClick: (ids: string[]) => handleBulkAction("dismiss", ids) },
    { label: t("common.delete"), variant: "destructive" as const, onClick: (ids: string[]) => handleBulkAction("delete", ids) },
  ]

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  // Reset page if current exceeds total after deletion.
  useEffect(() => {
    if (totalPages > 0 && page >= totalPages) setPage(totalPages - 1)
  }, [totalPages, page])

  const statusBadgeVariant = (status: string) => {
    switch (status) {
      case "pending":
        return "secondary"
      case "reviewed":
        return "default"
      case "dismissed":
        return "outline"
      case "actioned":
        return "destructive"
      default:
        return "outline"
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t('adminReports.title')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('adminReports.reportsCount', { count: total })}
        </p>
      </div>

      <Input
        placeholder={t('adminReports.searchPlaceholder')}
        value={search}
        onChange={(e) => {
          setSearch(e.target.value)
          setPage(0)
        }}
        className="max-w-sm"
      />

      <div className="flex gap-1">
        {STATUS_TABS.map((tab) => (
          <Button
            key={tab.value}
            size="sm"
            variant={statusFilter === tab.value ? "default" : "outline"}
            onClick={() => {
              setStatusFilter(tab.value)
              setPage(0)
            }}
          >
            {tab.label}
          </Button>
        ))}
      </div>

      {isLoading ? (
        <Card>
          <CardContent className="p-0">
            <TableSkeleton rows={5} cols={7} />
          </CardContent>
        </Card>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error?.message}</p>
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <SelectableTable
                items={data?.reports ?? []}
                getKey={(r) => r.id}
                columns={
                  <>
                    <TableHead>{t('adminReports.targetType')}</TableHead>
                    <TableHead>{t('adminReports.targetId')}</TableHead>
                    <TableHead>{t('adminReports.reason')}</TableHead>
                    <TableHead>{t('adminReports.reporter')}</TableHead>
                    <SortableHeader field="status" label={t('adminReports.status')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="created_at" label={t('adminReports.createdAt')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <TableHead className="text-right">{t('adminAgents.actions')}</TableHead>
                  </>
                }
                renderRow={(report) => (
                  <>
                    <TableCell>
                      <Badge variant="outline">{report.target_type}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs max-w-[120px] truncate">
                      {report.target_id}
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate">{report.reason}</TableCell>
                    <TableCell className="font-mono text-xs max-w-[120px] truncate">
                      {report.reporter_id}
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusBadgeVariant(report.status)}>
                        {report.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {new Date(report.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      {report.status === "pending" && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleStatusChange(report.id, "reviewed")}
                          >
                            {t('adminReports.review')}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleStatusChange(report.id, "dismissed")}
                          >
                            {t('adminReports.dismiss')}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleStatusChange(report.id, "actioned")}
                          >
                            {t('adminReports.action')}
                          </Button>
                        </>
                      )}
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-destructive"
                        onClick={() => setDeleteTarget(report.id)}
                      >
                        {t('common.delete')}
                      </Button>
                    </TableCell>
                  </>
                )}
                bulkActions={reportBulkActions}
                emptyMessage={t('adminReports.noReports')}
              />
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

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title={t('common.confirmDelete')}
        description={t('common.deleteConfirmation')}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
