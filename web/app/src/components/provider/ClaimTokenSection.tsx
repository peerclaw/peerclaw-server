import { useState, useEffect, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Copy, Check, Key, Loader2, RefreshCw } from "lucide-react"
import { useClaimTokens, useGenerateClaimToken } from "@/hooks/use-provider"

export function ClaimTokenSection() {
  const { data, loading, error, refetch } = useClaimTokens()
  const { generate } = useGenerateClaimToken()

  const [agentName, setAgentName] = useState("")
  const [generatedCode, setGeneratedCode] = useState<string | null>(null)
  const [expiresAt, setExpiresAt] = useState<string | null>(null)
  const [remaining, setRemaining] = useState<number>(0)
  const [copied, setCopied] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [genError, setGenError] = useState<string | null>(null)

  // Countdown timer
  useEffect(() => {
    if (!expiresAt) return
    const tick = () => {
      const diff = Math.max(
        0,
        Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000)
      )
      setRemaining(diff)
      if (diff <= 0) {
        setGeneratedCode(null)
        setExpiresAt(null)
      }
    }
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [expiresAt])

  const handleGenerate = useCallback(async () => {
    if (!agentName.trim()) {
      setGenError("Please enter an agent name")
      return
    }
    setGenerating(true)
    setGenError(null)
    try {
      const res = await generate({
        agent_name: agentName.trim(),
        protocols: ["a2a"],
      })
      setGeneratedCode(res.token)
      setExpiresAt(res.expires_at)
      setCopied(false)
      refetch()
    } catch (e) {
      setGenError(e instanceof Error ? e.message : "Failed to generate token")
    } finally {
      setGenerating(false)
    }
  }, [agentName, generate, refetch])

  const prompt = generatedCode
    ? `curl -fsSL https://peerclaw.ai/install.sh | sh\npeerclaw agent claim --token ${generatedCode}`
    : ""

  const handleCopy = useCallback(() => {
    if (!prompt) return
    navigator.clipboard.writeText(prompt)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [prompt])

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}:${s.toString().padStart(2, "0")}`
  }

  const statusVariant = (status: string) => {
    switch (status) {
      case "pending":
        return "secondary" as const
      case "claimed":
        return "default" as const
      default:
        return "outline" as const
    }
  }

  const statusLabel = (status: string, expiresAt: string) => {
    if (status === "pending" && new Date(expiresAt) < new Date()) {
      return "expired"
    }
    return status
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base">Claim Tokens</CardTitle>
          <p className="text-sm text-muted-foreground mt-0.5">
            Generate a one-time code to pair an Agent with your account
          </p>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Agent name input + generate button */}
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <label className="text-sm font-medium mb-1 block">
              Agent Name
            </label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleGenerate()}
              placeholder="e.g., my-research-agent"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <Button
            size="sm"
            onClick={handleGenerate}
            disabled={generating}
            className="shrink-0"
          >
            {generating ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Key className="size-4" />
            )}
            {generating ? "Generating..." : "Generate Token"}
          </Button>
        </div>

        {genError && (
          <p className="text-sm text-destructive">{genError}</p>
        )}

        {/* Generated prompt display */}
        {generatedCode && (
          <div className="rounded-lg border border-primary/30 bg-primary/5 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">
                Send this prompt to your Agent:
              </p>
              <span className="text-sm text-muted-foreground tabular-nums">
                {formatTime(remaining)}
              </span>
            </div>
            <pre className="text-sm font-mono bg-background rounded-md p-3 overflow-x-auto whitespace-pre-wrap break-all border">
              {prompt}
            </pre>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCopy}
            >
              {copied ? (
                <Check className="size-4 text-emerald-500" />
              ) : (
                <Copy className="size-4" />
              )}
              {copied ? "Copied" : "Copy Prompt"}
            </Button>
          </div>
        )}

        {/* Token history table */}
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading tokens...</p>
        ) : error ? (
          <p className="text-sm text-destructive">{error}</p>
        ) : data && data.tokens && data.tokens.length > 0 ? (
          <div>
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-medium text-muted-foreground">
                Token History
              </h4>
              <Button variant="ghost" size="sm" onClick={refetch}>
                <RefreshCw className="size-3" />
              </Button>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Code</TableHead>
                  <TableHead>Agent Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.tokens.slice(0, 10).map((t) => {
                  const displayStatus = statusLabel(t.status, t.expires_at)
                  return (
                    <TableRow key={t.id}>
                      <TableCell className="font-mono text-xs">
                        {t.code}
                      </TableCell>
                      <TableCell className="text-sm">
                        {t.agent_name || "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusVariant(displayStatus)}>
                          {displayStatus}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs">
                        {new Date(t.created_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            No tokens generated yet. Enter an agent name and click "Generate Token" to create a pairing code.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
