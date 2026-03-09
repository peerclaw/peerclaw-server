import type { LucideIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {change !== undefined && (
          <div className="flex items-center gap-1 mt-1">
            {change >= 0 ? (
              <TrendingUp className="size-3 text-emerald-500" />
            ) : (
              <TrendingDown className="size-3 text-red-500" />
            )}
            <span
              className={`text-xs font-medium ${
                change >= 0 ? "text-emerald-500" : "text-red-500"
              }`}
            >
              {change >= 0 ? "+" : ""}
              {change.toFixed(1)}%
            </span>
            <span className="text-xs text-muted-foreground">{t('common.vsLastPeriod')}</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
