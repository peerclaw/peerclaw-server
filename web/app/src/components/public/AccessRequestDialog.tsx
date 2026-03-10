import { useState } from "react"
import { useTranslation } from "react-i18next"
import { submitAccessRequest } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"
import { Lock, Loader2 } from "lucide-react"

interface AccessRequestDialogProps {
  agentId: string
  onSubmitted: () => void
}

export function AccessRequestDialog({ agentId, onSubmitted }: AccessRequestDialogProps) {
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [open, setOpen] = useState(false)
  const [message, setMessage] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async () => {
    if (!accessToken) return
    setSubmitting(true)
    setError(null)
    try {
      await submitAccessRequest(agentId, message, accessToken)
      setOpen(false)
      setMessage("")
      onSubmitted()
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to submit request")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        <Lock className="size-4" />
        {t('accessRequest.requestAccess')}
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-lg">
            <h2 className="text-lg font-semibold">{t('accessRequest.submitRequest')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('accessRequest.submitRequestDesc')}
            </p>

            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={t('accessRequest.messagePlaceholder')}
              rows={3}
              className="mt-4 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
            />

            {error && <p className="mt-2 text-sm text-destructive">{error}</p>}

            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => {
                  setOpen(false)
                  setError(null)
                }}
                className="rounded-md border border-border px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleSubmit}
                disabled={submitting}
                className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {submitting && <Loader2 className="size-3.5 animate-spin" />}
                {t('common.submit')}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
