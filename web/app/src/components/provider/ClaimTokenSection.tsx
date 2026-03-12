import { useState, useEffect, useCallback } from "react"
import { useTranslation } from "react-i18next"
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
import { Copy, Check, Key, Loader2, RefreshCw, Eye } from "lucide-react"
import { useClaimTokens, useGenerateClaimToken } from "@/hooks/use-provider"

export function ClaimTokenSection() {
  const { t } = useTranslation()
  const { data, loading, error, refetch } = useClaimTokens()
  const { generate } = useGenerateClaimToken()

  const [agentName, setAgentName] = useState("")
  const [generatedCode, setGeneratedCode] = useState<string | null>(null)
  const [expiresAt, setExpiresAt] = useState<string | null>(null)
  const [remaining, setRemaining] = useState<number>(0)
  const [copied, setCopied] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [genError, setGenError] = useState<string | null>(null)
  const [viewingToken, setViewingToken] = useState<{ code: string; agent_name: string } | null>(null)
  const [viewCopied, setViewCopied] = useState(false)

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
      setGenError(t('claim.enterAgentName'))
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
  }, [agentName, generate, refetch, t])

  const buildPrompt = (token: string, name: string) => `Please register me on PeerClaw (https://peerclaw.ai) — an open-source identity & trust platform for AI Agents.

## Step 1: Install the PeerClaw CLI

\`\`\`bash
curl -fsSL https://peerclaw.ai/install.sh | sh
\`\`\`

This downloads the \`peerclaw\` binary from GitHub releases: https://github.com/peerclaw/peerclaw-cli/releases
You can inspect the script first: \`curl -fsSL https://peerclaw.ai/install.sh\`

## Step 2: Register as "${name}"

\`\`\`bash
peerclaw agent claim --token ${token} --server ${window.location.origin}
\`\`\`

This command will:
- Generate an Ed25519 keypair (saved to \`./agent.key\`)
- Sign the token with your private key (proves key ownership)
- Register with PeerClaw as "${name}" on ${window.location.origin}

⏰ Token expires in 30 minutes — please run immediately.

## Step 3: After registration

Save these details from the output for future use:
- **Agent ID** — your unique identifier on the platform
- **Keypair file** (\`./agent.key\`) — keep this safe, it's your cryptographic identity

Set the server once so you don't need \`--server\` every time:
\`\`\`bash
peerclaw config set server ${window.location.origin}
\`\`\`

Useful follow-up commands:
\`\`\`bash
peerclaw agent get <your-agent-id>                # verify registration
peerclaw agent heartbeat <your-agent-id> --status active  # stay discoverable
peerclaw invoke <other-agent-id> --message "Hello"  # talk to other agents
peerclaw mcp serve                                  # run as MCP tool server
\`\`\`

Full documentation: https://github.com/peerclaw/peerclaw/blob/main/docs/GUIDE.md`

  const prompt = generatedCode
    ? buildPrompt(generatedCode, agentName.trim())
    : ""

  const viewPrompt = viewingToken
    ? buildPrompt(viewingToken.code, viewingToken.agent_name)
    : ""

  const handleCopy = useCallback(() => {
    if (!prompt) return
    navigator.clipboard.writeText(prompt)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [prompt])

  const handleViewCopy = useCallback(() => {
    if (!viewPrompt) return
    navigator.clipboard.writeText(viewPrompt)
    setViewCopied(true)
    setTimeout(() => setViewCopied(false), 2000)
  }, [viewPrompt])

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


  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base">{t('claim.title')}</CardTitle>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('claim.description')}
          </p>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Agent name input + generate button */}
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <label className="text-sm font-medium mb-1 block">
              {t('claim.agentName')}
            </label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleGenerate()}
              placeholder={t('claim.agentNamePlaceholder')}
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
            {generating ? t('claim.generating') : t('claim.generateToken')}
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
                {t('claim.sendPrompt')}
              </p>
              <span className="text-sm text-muted-foreground tabular-nums">
                {formatTime(remaining)}
              </span>
            </div>
            <pre className="text-sm font-mono bg-background rounded-md p-3 overflow-x-auto whitespace-pre-wrap break-all border max-h-64 overflow-y-auto">
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
              {copied ? t('claim.copied') : t('claim.copyPrompt')}
            </Button>
          </div>
        )}

        {/* View prompt panel (from token history) */}
        {viewingToken && (
          <div className="rounded-lg border border-primary/30 bg-primary/5 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">
                {t('claim.sendPrompt')}
              </p>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { setViewingToken(null); setViewCopied(false) }}
              >
                {t('common.close')}
              </Button>
            </div>
            <pre className="text-sm font-mono bg-background rounded-md p-3 overflow-x-auto whitespace-pre-wrap break-all border max-h-64 overflow-y-auto">
              {viewPrompt}
            </pre>
            <Button
              variant="outline"
              size="sm"
              onClick={handleViewCopy}
            >
              {viewCopied ? (
                <Check className="size-4 text-emerald-500" />
              ) : (
                <Copy className="size-4" />
              )}
              {viewCopied ? t('claim.copied') : t('claim.copyPrompt')}
            </Button>
          </div>
        )}

        {/* Token history table */}
        {loading ? (
          <p className="text-sm text-muted-foreground">{t('claim.loadingTokens')}</p>
        ) : error ? (
          <p className="text-sm text-destructive">{error}</p>
        ) : data && data.tokens && data.tokens.filter((tk) => !(tk.status === "pending" && new Date(tk.expires_at) < new Date())).length > 0 ? (
          <div>
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-medium text-muted-foreground">
                {t('claim.tokenHistory')}
              </h4>
              <Button variant="ghost" size="sm" onClick={refetch}>
                <RefreshCw className="size-3" />
              </Button>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('claim.code')}</TableHead>
                  <TableHead>{t('claim.agentName')}</TableHead>
                  <TableHead>{t('claim.status')}</TableHead>
                  <TableHead>{t('apiKeys.created')}</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.tokens
                  .filter((tk) => !(tk.status === "pending" && new Date(tk.expires_at) < new Date()))
                  .slice(0, 10)
                  .map((tk) => {
                  const isPending = tk.status === "pending" && new Date(tk.expires_at) >= new Date()
                  return (
                    <TableRow key={tk.id}>
                      <TableCell className="font-mono text-xs">
                        {tk.code}
                      </TableCell>
                      <TableCell className="text-sm">
                        {tk.agent_name || "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusVariant(tk.status)}>
                          {tk.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs">
                        {new Date(tk.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        {isPending && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => { setViewingToken({ code: tk.code, agent_name: tk.agent_name || "" }); setViewCopied(false) }}
                          >
                            <Eye className="size-3.5" />
                            {t('claim.viewPrompt')}
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            {t('claim.noTokens')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
