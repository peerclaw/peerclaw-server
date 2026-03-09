import { useState } from "react"
import { useTranslation } from "react-i18next"
import { submitReport } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"
import { Flag, X } from "lucide-react"

interface ReportDialogProps {
  targetType: "agent" | "review"
  targetId: string
}

export function ReportDialog({ targetType, targetId }: ReportDialogProps) {
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState("")
  const [details, setDetails] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState("")

  const REASONS = [
    t('report.spam'),
    t('report.malicious'),
    t('report.inappropriate'),
    t('report.impersonation'),
    t('report.other'),
  ]

  if (!accessToken) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!reason) return
    setSubmitting(true)
    setError("")
    try {
      await submitReport(targetType, targetId, reason, details, accessToken)
      setDone(true)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive"
      >
        <Flag className="size-3" />
        {t('report.report')}
      </button>
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-full max-w-md rounded-lg border border-border bg-card p-6">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-semibold">{t('report.title', { type: targetType })}</h3>
          <button onClick={() => setOpen(false)}>
            <X className="size-4 text-muted-foreground" />
          </button>
        </div>

        {done ? (
          <div>
            <p className="text-sm text-emerald-400">{t('report.submitted')}</p>
            <button
              onClick={() => { setOpen(false); setDone(false) }}
              className="mt-3 rounded-lg bg-secondary px-4 py-1.5 text-xs"
            >
              {t('common.close')}
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs text-muted-foreground">{t('report.reason')}</label>
              <select
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="">{t('report.selectReason')}</option>
                {REASONS.map((r) => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs text-muted-foreground">
                {t('report.details')}
              </label>
              <textarea
                value={details}
                onChange={(e) => setDetails(e.target.value)}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
                rows={3}
              />
            </div>
            {error && <p className="text-xs text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="rounded-lg bg-secondary px-4 py-1.5 text-xs"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={submitting || !reason}
                className="rounded-lg bg-destructive px-4 py-1.5 text-xs text-destructive-foreground disabled:opacity-50"
              >
                {submitting ? t('report.submitting') : t('report.submitReport')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
