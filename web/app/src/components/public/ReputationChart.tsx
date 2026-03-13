import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import type { ReputationEvent } from "@/api/types"

function useCssColor(varName: string): string {
  const [color, setColor] = useState("")
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current ?? document.documentElement
    const resolve = () => {
      const val = getComputedStyle(el).getPropertyValue(varName).trim()
      setColor(val || "#888")
    }
    resolve()
    const observer = new MutationObserver(resolve)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })
    return () => observer.disconnect()
  }, [varName])

  return color
}

export function ReputationChart({ events }: { events: ReputationEvent[] }) {
  const { t } = useTranslation()
  const chartColor = useCssColor("--chart-1")
  const borderColor = useCssColor("--border")
  const mutedColor = useCssColor("--muted-foreground")
  const cardColor = useCssColor("--card")
  const fgColor = useCssColor("--foreground")

  // Events come newest first, reverse for chronological display.
  const data = [...(events ?? [])].reverse().map((e) => ({
    time: new Date(e.created_at).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }),
    score: Math.round(e.score_after * 100),
    type: e.event_type,
  }))

  if (data.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-muted-foreground text-sm">
        {t('reputation.noEvents')}
      </div>
    )
  }

  // Calculate tick interval to show at most ~6 labels
  const tickInterval = data.length <= 6 ? 0 : Math.ceil(data.length / 6) - 1

  return (
    <ResponsiveContainer width="100%" height={240}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="repChartGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={chartColor} stopOpacity={0.3} />
            <stop offset="100%" stopColor={chartColor} stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke={borderColor} />
        <XAxis
          dataKey="time"
          tick={{ fontSize: 11, fill: mutedColor }}
          tickLine={false}
          interval={tickInterval}
          angle={-30}
          textAnchor="end"
          height={50}
        />
        <YAxis
          domain={[0, 100]}
          tick={{ fontSize: 11, fill: mutedColor }}
          tickLine={false}
          width={35}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: cardColor,
            border: `1px solid ${borderColor}`,
            borderRadius: 8,
            fontSize: 12,
          }}
          labelStyle={{ color: fgColor }}
        />
        <Area
          type="monotone"
          dataKey="score"
          stroke={chartColor}
          strokeWidth={2}
          fill="url(#repChartGradient)"
          dot={false}
          activeDot={{ r: 5, fill: chartColor, strokeWidth: 2, stroke: cardColor }}
          connectNulls
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
