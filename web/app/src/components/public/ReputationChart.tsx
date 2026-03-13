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

export function ReputationChart({ events }: { events: ReputationEvent[] }) {
  const { t } = useTranslation()

  // Events come newest first, reverse for chronological display.
  const data = [...(events ?? [])].reverse().map((e) => ({
    time: new Date(e.created_at).toLocaleDateString(undefined, {
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

  return (
    <ResponsiveContainer width="100%" height={240}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="chartGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="hsl(var(--chart-1))" stopOpacity={0.3} />
            <stop offset="100%" stopColor="hsl(var(--chart-1))" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
        <XAxis
          dataKey="time"
          tick={{ fontSize: 11, fill: "hsl(var(--muted-foreground))" }}
          tickLine={false}
        />
        <YAxis
          domain={[0, 100]}
          tick={{ fontSize: 11, fill: "hsl(var(--muted-foreground))" }}
          tickLine={false}
          width={35}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: "hsl(var(--card))",
            border: "1px solid hsl(var(--border))",
            borderRadius: 8,
            fontSize: 12,
          }}
          labelStyle={{ color: "hsl(var(--foreground))" }}
        />
        <Area
          type="monotone"
          dataKey="score"
          stroke="hsl(var(--chart-1))"
          strokeWidth={2.5}
          fill="url(#chartGradient)"
          dot={{ r: 4, fill: "hsl(var(--chart-1))", strokeWidth: 0 }}
          activeDot={{ r: 6, fill: "hsl(var(--chart-1))", strokeWidth: 2, stroke: "hsl(var(--card))" }}
          connectNulls
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
