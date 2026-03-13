import { useTranslation } from "react-i18next"
import {
  LineChart,
  Line,
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
      <LineChart data={data}>
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
        <Line
          type="monotone"
          dataKey="score"
          stroke="hsl(var(--chart-1))"
          strokeWidth={2}
          dot={{ r: 4, fill: "hsl(var(--chart-1))", strokeWidth: 0 }}
          activeDot={{ r: 6, fill: "hsl(var(--chart-1))", strokeWidth: 2, stroke: "hsl(var(--card))" }}
          connectNulls
        />
      </LineChart>
    </ResponsiveContainer>
  )
}
