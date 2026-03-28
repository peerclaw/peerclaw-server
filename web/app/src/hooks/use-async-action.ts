import { useState, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

interface UseAsyncActionOptions {
  successMessage?: string
  onSuccess?: () => void
}

export function useAsyncAction<Args extends unknown[]>(
  action: (...args: Args) => Promise<unknown>,
  options?: UseAsyncActionOptions
) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const execute = useCallback(
    async (...args: Args) => {
      setLoading(true)
      try {
        await action(...args)
        if (options?.successMessage) {
          toast.success(options.successMessage)
        }
        options?.onSuccess?.()
      } catch (e) {
        toast.error(
          e instanceof Error ? e.message : t("toast.operationFailed")
        )
      } finally {
        setLoading(false)
      }
    },
    [action, options, t]
  )

  return { execute, loading }
}
