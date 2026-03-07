import { ShieldCheck } from "lucide-react"

export function TrustedBadge() {
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400"
      title="Trusted: verified agent with high reputation"
    >
      <ShieldCheck className="size-3" />
      Trusted
    </span>
  )
}
