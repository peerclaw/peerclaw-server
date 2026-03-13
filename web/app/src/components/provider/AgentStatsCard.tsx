import type { LucideIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { TrendingUp, TrendingDown } from "lucide-react"

interface AgentStatsCardProps {
  title: string
  value: string | number
  change?: number
  icon: LucideIcon
}

export function AgentStatsCard({ title, value, change, icon: Icon }: AgentStatsCardProps) {
  const { t } = useTranslation()

  return (
    <div className="group relative overflow-hidden rounded-xl border border-border/60 bg-card p-5 transition-all duration-300 hover:border-primary/20 hover:shadow-[0_0_20px_oklch(0.72_0.15_192_/_0.05)]">
      {/* Subtle gradient accent at top */}
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-primary/30 to-transparent opacity-0 transition-opacity group-hover:opacity-100" />

      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-muted-foreground">{title}</p>
        <div className="flex size-8 items-center justify-center rounded-lg bg-primary/8 transition-colors group-hover:bg-primary/12">
          <Icon className="size-4 text-primary/70" />
        </div>
      </div>
      <div className="mt-3">
        <p className="text-2xl font-bold tracking-tight">{value}</p>
        {change !== undefined && (
          <div className="flex items-center gap-1 mt-1.5">
            {change >= 0 ? (
              <TrendingUp className="size-3 text-emerald-500" />
            ) : (
              <TrendingDown className="size-3 text-red-400" />
            )}
            <span
              className={`text-xs font-medium ${
                change >= 0 ? "text-emerald-500" : "text-red-400"
              }`}
            >
              {change >= 0 ? "+" : ""}
              {change.toFixed(1)}%
            </span>
            <span className="text-xs text-muted-foreground">{t('common.vsLastPeriod')}</span>
          </div>
        )}
      </div>
    </div>
  )
}
