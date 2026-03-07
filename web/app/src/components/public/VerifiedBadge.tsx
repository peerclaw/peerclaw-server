import { CheckCircle } from "lucide-react"

export function VerifiedBadge({ className }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400 ${className ?? ""}`}
      title="Verified endpoint"
    >
      <CheckCircle className="size-3" />
      Verified
    </span>
  )
}
