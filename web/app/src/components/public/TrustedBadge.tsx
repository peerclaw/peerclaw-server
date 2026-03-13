import { useTranslation } from "react-i18next"
import { ShieldCheck } from "lucide-react"

export function TrustedBadge() {
  const { t } = useTranslation()
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary ring-1 ring-primary/20"
      title={t('badge.trustedTooltip')}
    >
      <ShieldCheck className="size-3" />
      {t('badge.trusted')}
    </span>
  )
}
