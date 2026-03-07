function scoreColor(score: number): string {
  if (score >= 0.7) return "text-emerald-400"
  if (score >= 0.4) return "text-yellow-400"
  return "text-red-400"
}

function barColor(score: number): string {
  if (score >= 0.7) return "bg-emerald-500"
  if (score >= 0.4) return "bg-yellow-500"
  return "bg-red-500"
}

export function ReputationMeter({
  score,
  size = "md",
}: {
  score: number
  size?: "sm" | "md" | "lg"
}) {
  const pct = Math.round(score * 100)

  const textSize = size === "lg" ? "text-2xl" : size === "md" ? "text-lg" : "text-sm"
  const barHeight = size === "lg" ? "h-3" : size === "md" ? "h-2" : "h-1.5"

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-baseline gap-1">
        <span className={`font-bold tabular-nums ${textSize} ${scoreColor(score)}`}>
          {pct}
        </span>
        <span className="text-xs text-muted-foreground">/100</span>
      </div>
      <div className={`w-full rounded-full bg-muted ${barHeight}`}>
        <div
          className={`rounded-full ${barHeight} ${barColor(score)} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}
