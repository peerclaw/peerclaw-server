import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Button } from "@/components/ui/button"
import { ArrowLeft, Copy } from "lucide-react"
import { useNavigate } from "react-router-dom"
import type { Agent } from "@/api/types"

interface Props {
  agent: Agent
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

export function AgentCardView({ agent }: Props) {
  const navigate = useNavigate()

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  return (
    <div className="space-y-6">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate("/agents")}
        className="gap-1.5"
      >
        <ArrowLeft className="size-4" />
        Back to Agents
      </Button>

      {/* Agent Info */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className={`size-3 rounded-full ${statusColor(agent.status)}`} />
            <CardTitle>{agent.name}</CardTitle>
            <Badge variant="secondary">{agent.status}</Badge>
          </div>
          {agent.description && (
            <p className="text-sm text-muted-foreground mt-1">{agent.description}</p>
          )}
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Capabilities */}
          {agent.capabilities && agent.capabilities.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2">
                Capabilities
              </h4>
              <div className="flex flex-wrap gap-1.5">
                {agent.capabilities.map((c) => (
                  <Badge key={c} variant="secondary">
                    {c}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Protocols */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2">
              Protocols
            </h4>
            <div className="flex gap-1.5">
              {agent.protocols.map((p) => (
                <Badge key={p} variant="outline">
                  {p.toUpperCase()}
                </Badge>
              ))}
            </div>
          </div>

          {/* Endpoint */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground mb-2">
              Endpoint
            </h4>
            <code className="text-sm bg-muted px-2 py-1 rounded">
              {agent.endpoint.url}
            </code>
          </div>
        </CardContent>
      </Card>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Metadata</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {/* Public Key */}
          {agent.public_key && (
            <div className="flex items-center justify-between">
              <div>
                <h4 className="text-xs font-medium text-muted-foreground">
                  Public Key
                </h4>
                <code className="text-xs">
                  {agent.public_key.slice(0, 20)}...{agent.public_key.slice(-8)}
                </code>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => copyToClipboard(agent.public_key)}
              >
                <Copy className="size-3.5" />
              </Button>
            </div>
          )}

          <Separator />

          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <h4 className="text-xs font-medium text-muted-foreground">Version</h4>
              <p className="text-sm">{agent.version || "—"}</p>
            </div>
            <div>
              <h4 className="text-xs font-medium text-muted-foreground">
                Registered At
              </h4>
              <p className="text-sm">
                {new Date(agent.registered_at).toLocaleString()}
              </p>
            </div>
            <div>
              <h4 className="text-xs font-medium text-muted-foreground">
                Last Heartbeat
              </h4>
              <p className="text-sm">
                {new Date(agent.last_heartbeat).toLocaleString()}
              </p>
            </div>
            <div>
              <h4 className="text-xs font-medium text-muted-foreground">Transport</h4>
              <p className="text-sm">{agent.endpoint.transport || "—"}</p>
            </div>
          </div>

          {/* PeerClaw Extension */}
          {agent.peerclaw && (
            <>
              <Separator />
              <h4 className="text-xs font-medium text-muted-foreground">
                PeerClaw Extension
              </h4>
              <div className="grid gap-3 sm:grid-cols-2">
                {agent.peerclaw.nat_type && (
                  <div>
                    <h4 className="text-xs text-muted-foreground">NAT Type</h4>
                    <p className="text-sm">{agent.peerclaw.nat_type}</p>
                  </div>
                )}
                {agent.peerclaw.relay_preference && (
                  <div>
                    <h4 className="text-xs text-muted-foreground">
                      Relay Preference
                    </h4>
                    <p className="text-sm">{agent.peerclaw.relay_preference}</p>
                  </div>
                )}
                {agent.peerclaw.tags && agent.peerclaw.tags.length > 0 && (
                  <div className="sm:col-span-2">
                    <h4 className="text-xs text-muted-foreground mb-1">Tags</h4>
                    <div className="flex flex-wrap gap-1">
                      {agent.peerclaw.tags.map((t) => (
                        <Badge key={t} variant="secondary" className="text-[10px]">
                          {t}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
