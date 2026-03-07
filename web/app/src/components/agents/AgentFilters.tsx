import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search } from "lucide-react"

interface Props {
  search: string
  onSearchChange: (value: string) => void
  protocol: string
  onProtocolChange: (value: string) => void
  status: string
  onStatusChange: (value: string) => void
}

const protocols = ["all", "a2a", "mcp", "acp"]
const statuses = ["all", "online", "offline"]

export function AgentFilters({
  search,
  onSearchChange,
  protocol,
  onProtocolChange,
  status,
  onStatusChange,
}: Props) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
      <div className="relative flex-1 max-w-sm">
        <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
        <Input
          placeholder="Search agents..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9"
        />
      </div>

      <div className="flex gap-1">
        {protocols.map((p) => (
          <Button
            key={p}
            variant={protocol === p ? "default" : "secondary"}
            size="sm"
            onClick={() => onProtocolChange(p)}
          >
            {p === "all" ? "All" : p.toUpperCase()}
          </Button>
        ))}
      </div>

      <div className="flex gap-1">
        {statuses.map((s) => (
          <Button
            key={s}
            variant={status === s ? "default" : "secondary"}
            size="sm"
            onClick={() => onStatusChange(s)}
          >
            {s === "all" ? "All" : s.charAt(0).toUpperCase() + s.slice(1)}
          </Button>
        ))}
      </div>
    </div>
  )
}
