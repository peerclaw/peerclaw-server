import { useTranslation } from "react-i18next"
import type { TimeSeriesPoint } from "@/hooks/use-provider"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

interface AnalyticsChartProps {
  data: TimeSeriesPoint[]
  title?: string
}

export function AnalyticsChart({ data, title }: AnalyticsChartProps) {
  const { t } = useTranslation()

  const displayTitle = title ?? t('analyticsChart.invocationsOverTime')

  if (!data.length) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">{displayTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">{t('analyticsChart.noData')}</p>
        </CardContent>
      </Card>
    )
  }

  const maxCount = Math.max(...data.map((d) => d.count), 1)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{displayTitle}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-end gap-1 h-40">
          {data.map((point, i) => {
            const heightPercent = (point.count / maxCount) * 100
            const date = new Date(point.timestamp)
            const label = `${date.getMonth() + 1}/${date.getDate()}`

            return (
              <div
                key={i}
                className="flex flex-1 flex-col items-center gap-1 group"
              >
                {/* Tooltip on hover */}
                <div className="text-[10px] text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity">
                  {point.count}
                </div>
                {/* Bar */}
                <div className="w-full flex items-end justify-center" style={{ height: "100%" }}>
                  <div
                    className="w-full max-w-[28px] rounded-t bg-primary transition-all hover:bg-primary/80"
                    style={{ height: `${Math.max(heightPercent, 2)}%` }}
                  />
                </div>
                {/* Label */}
                <span className="text-[10px] text-muted-foreground">{label}</span>
              </div>
            )
          })}
        </div>
        <div className="flex items-center justify-between mt-2 text-xs text-muted-foreground">
          <span>
            {t('analyticsChart.total', { count: data.reduce((sum, d) => sum + d.count, 0) })}
          </span>
          <span>{t('analyticsChart.peak', { count: maxCount })}</span>
        </div>
      </CardContent>
    </Card>
  )
}
