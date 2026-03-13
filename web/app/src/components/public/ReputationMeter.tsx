function scoreColor(score: number): string {
  if (score >= 0.7) return "oklch(0.72 0.18 160)"
  if (score >= 0.4) return "oklch(0.8 0.15 85)"
  return "oklch(0.65 0.2 25)"
}

function scoreTextColor(score: number): string {
  if (score >= 0.7) return "text-emerald-400"
  if (score >= 0.4) return "text-amber-400"
  return "text-red-400"
}

export function ReputationMeter({
  score,
  size = "md",
}: {
  score: number
  size?: "sm" | "md" | "lg"
}) {
  const pct = Math.round(score * 100)

  const dims = {
    sm: { svgSize: 64, radius: 26, stroke: 4, fontSize: "text-sm", labelSize: "text-[9px]" },
    md: { svgSize: 80, radius: 32, stroke: 5, fontSize: "text-lg", labelSize: "text-[10px]" },
    lg: { svgSize: 112, radius: 46, stroke: 6, fontSize: "text-2xl", labelSize: "text-xs" },
  }
  const d = dims[size]
  const circumference = 2 * Math.PI * d.radius
  const offset = circumference * (1 - score)
  const color = scoreColor(score)
  const center = d.svgSize / 2

  return (
    <div className="flex flex-col items-center">
      <svg
        width={d.svgSize}
        height={d.svgSize}
        viewBox={`0 0 ${d.svgSize} ${d.svgSize}`}
        className="drop-shadow-[0_0_8px_oklch(0.72_0.15_192_/_0.15)]"
      >
        {/* Background track */}
        <circle
          cx={center}
          cy={center}
          r={d.radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={d.stroke}
          className="text-border"
        />
        {/* Progress arc */}
        <circle
          cx={center}
          cy={center}
          r={d.radius}
          fill="none"
          stroke={color}
          strokeWidth={d.stroke}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform={`rotate(-90 ${center} ${center})`}
          className="transition-all duration-1000 ease-out"
          style={{
            filter: `drop-shadow(0 0 4px ${color})`,
          }}
        />
        {/* Score text */}
        <text
          x={center}
          y={center - 2}
          textAnchor="middle"
          dominantBaseline="central"
          className={`font-bold tabular-nums fill-current ${scoreTextColor(score)} ${d.fontSize}`}
          style={{ fontFamily: "'Outfit', sans-serif" }}
        >
          {pct}
        </text>
        <text
          x={center}
          y={center + (size === "lg" ? 18 : size === "md" ? 14 : 11)}
          textAnchor="middle"
          dominantBaseline="central"
          className={`fill-current text-muted-foreground ${d.labelSize}`}
          style={{ fontFamily: "'Outfit', sans-serif" }}
        >
          /100
        </text>
      </svg>
    </div>
  )
}
