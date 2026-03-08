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
    setGenerating(true)
    setGenError(null)
    try {
      const res = await generate()
      setGeneratedCode(res.token)
      setExpiresAt(res.expires_at)
      setCopied(false)
      refetch()
    } catch (e) {
      setGenError(e instanceof Error ? e.message : "Failed to generate token")
    } finally {
      setGenerating(false)
    }
  }, [generate, refetch])

  const handleCopy = useCallback(() => {
    if (!generatedCode) return
    navigator.clipboard.writeText(generatedCode)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [generatedCode])

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
        <Button
          size="sm"
          onClick={handleGenerate}
          disabled={generating}
        >
          {generating ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Key className="size-4" />
          )}
          {generating ? "Generating..." : "Generate Token"}
        </Button>
      </CardHeader>

      <CardContent className="space-y-4">
        {genError && (
          <p className="text-sm text-destructive">{genError}</p>
        )}

        {/* Generated code display */}
        {generatedCode && (
          <div className="flex items-center gap-3 rounded-lg border border-primary/30 bg-primary/5 p-4">
            <code className="text-xl font-mono font-bold tracking-widest text-primary">
              {generatedCode}
            </code>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCopy}
              className="shrink-0"
            >
              {copied ? (
                <Check className="size-4 text-emerald-500" />
              ) : (
                <Copy className="size-4" />
              )}
              {copied ? "Copied" : "Copy"}
            </Button>
            <span className="text-sm text-muted-foreground ml-auto tabular-nums">
              {formatTime(remaining)}
            </span>
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
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Agent</TableHead>
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
                      <TableCell>
                        <Badge variant={statusVariant(displayStatus)}>
                          {displayStatus}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs">
                        {new Date(t.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs font-mono">
                        {t.agent_id
                          ? t.agent_id.slice(0, 8) + "..."
                          : "-"}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            No tokens generated yet. Click "Generate Token" to create a pairing code.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
