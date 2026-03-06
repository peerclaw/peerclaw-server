import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { useNavigate } from "react-router-dom"
import type { Agent } from "@/api/types"

interface Props {
  agents: Agent[]
}

function statusColor(status: string) {
  switch (status) {
    case "online":
      return "bg-emerald-500"
    case "offline":
      return "bg-zinc-500"
    case "degraded":
      return "bg-amber-500"
    default:
      return "bg-zinc-500"
  }
}

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return "just now"
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

export function AgentTable({ agents }: Props) {
  const navigate = useNavigate()

  if (agents.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center rounded-md border border-dashed border-border">
        <p className="text-sm text-muted-foreground">No agents found</p>
      </div>
    )
  }

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10">Status</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Capabilities</TableHead>
            <TableHead>Protocols</TableHead>
            <TableHead className="text-right">Last Heartbeat</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {agents.map((agent) => (
            <TableRow
              key={agent.id}
              className="cursor-pointer"
              onClick={() => navigate(`/agents/${agent.id}`)}
            >
              <TableCell>
                <span
                  className={`inline-block size-2.5 rounded-full ${statusColor(
                    agent.status
                  )}`}
                />
              </TableCell>
              <TableCell className="font-medium">{agent.name}</TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-1">
                  {(agent.capabilities ?? []).slice(0, 3).map((c) => (
                    <Badge key={c} variant="secondary" className="text-[10px]">
                      {c}
                    </Badge>
                  ))}
                  {(agent.capabilities ?? []).length > 3 && (
                    <Badge variant="secondary" className="text-[10px]">
                      +{agent.capabilities.length - 3}
                    </Badge>
                  )}
                </div>
              </TableCell>
              <TableCell>
                <div className="flex gap-1">
                  {(agent.protocols ?? []).map((p) => (
                    <Badge key={p} variant="outline" className="text-[10px]">
                      {p.toUpperCase()}
                    </Badge>
                  ))}
                </div>
              </TableCell>
              <TableCell className="text-right text-muted-foreground text-sm">
                {timeAgo(agent.last_heartbeat)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
