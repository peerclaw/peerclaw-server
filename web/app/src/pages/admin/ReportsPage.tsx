import { useState, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useAdminReports, useAdminMutations } from "@/hooks/use-admin"
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

const PAGE_SIZE = 20

export function ReportsPage() {
  const { t } = useTranslation()
  const [statusFilter, setStatusFilter] = useState("")
  const [page, setPage] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const STATUS_TABS = [
    { label: t('common.all'), value: "" },
    { label: t('adminReports.pending'), value: "pending" },
    { label: t('adminReports.reviewed'), value: "reviewed" },
    { label: t('adminReports.dismissed'), value: "dismissed" },
    { label: t('adminReports.actioned'), value: "actioned" },
  ]

  const { data, loading, error, refetch } = useAdminReports(
    statusFilter || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )
  const { updateReport, deleteReport } = useAdminMutations()

  const handleStatusChange = useCallback(
    async (id: string, newStatus: string) => {
      try {
        await updateReport(id, newStatus)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to update report")
      }
    },
    [updateReport, refetch]
  )

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteReport(id)
        setConfirmDelete(null)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to delete report")
      }
    },
    [deleteReport, refetch]
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

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

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('adminReports.loadingReports')}</p>
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
                    <TableHead>{t('adminReports.targetType')}</TableHead>
                    <TableHead>{t('adminReports.targetId')}</TableHead>
                    <TableHead>{t('adminReports.reason')}</TableHead>
                    <TableHead>{t('adminReports.reporter')}</TableHead>
                    <TableHead>{t('adminReports.status')}</TableHead>
                    <TableHead>{t('adminReports.createdAt')}</TableHead>
                    <TableHead className="text-right">{t('adminAgents.actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(data?.reports ?? []).map((report) => (
                    <TableRow key={report.id}>
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
                        {confirmDelete === report.id ? (
                          <span className="inline-flex gap-1">
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleDelete(report.id)}
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
                          <>
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
                              onClick={() => setConfirmDelete(report.id)}
                            >
                              {t('common.delete')}
                            </Button>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(data?.reports ?? []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                        {t('adminReports.noReports')}
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
