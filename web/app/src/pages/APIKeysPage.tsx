import { useState, useEffect, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useAuth } from "@/hooks/use-auth"
import { createAPIKey, listAPIKeys, revokeAPIKey } from "@/api/auth"
import type { APIKey } from "@/api/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import { Plus, Copy, Check, Trash2, KeyRound, AlertTriangle } from "lucide-react"

export function APIKeysPage() {
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // New key creation
  const [showCreate, setShowCreate] = useState(false)
  const [newKeyName, setNewKeyName] = useState("")
  const [creating, setCreating] = useState(false)
  const [newKeySecret, setNewKeySecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  // Revoke state
  const [revokingId, setRevokingId] = useState<string | null>(null)

  const loadKeys = useCallback(async () => {
    if (!accessToken) return
    try {
      setLoading(true)
      setError(null)
      const result = await listAPIKeys(accessToken)
      setKeys(result.api_keys ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load API keys")
    } finally {
      setLoading(false)
    }
  }, [accessToken])

  useEffect(() => {
    loadKeys()
  }, [loadKeys])

  const handleCreate = async () => {
    if (!accessToken || !newKeyName.trim()) return
    setCreating(true)
    setError(null)
    try {
      const result = await createAPIKey(accessToken, newKeyName.trim())
      setNewKeySecret(result.key)
      setNewKeyName("")
      // Reload the list to include the new key
      await loadKeys()
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create API key")
    } finally {
      setCreating(false)
    }
  }

  const handleRevoke = async (keyId: string) => {
    if (!accessToken) return
    const confirmed = window.confirm(
      t('apiKeys.revokeConfirm')
    )
    if (!confirmed) return

    setRevokingId(keyId)
    setError(null)
    try {
      await revokeAPIKey(accessToken, keyId)
      await loadKeys()
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to revoke API key")
    } finally {
      setRevokingId(null)
    }
  }

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback: select text
    }
  }

  const activeKeys = keys.filter((k) => !k.revoked)
  const revokedKeys = keys.filter((k) => k.revoked)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('apiKeys.title')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('apiKeys.description')}
          </p>
        </div>
        <Button size="sm" onClick={() => setShowCreate(true)} disabled={showCreate}>
          <Plus className="size-4" />
          {t('apiKeys.generateNew')}
        </Button>
      </div>

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {/* New key creation form */}
      {showCreate && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">{t('apiKeys.generateTitle')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {newKeySecret ? (
              <div className="space-y-3">
                <div className="flex items-center gap-2 p-3 rounded-md bg-amber-500/10 border border-amber-500/30">
                  <AlertTriangle className="size-4 text-amber-500 shrink-0" />
                  <p className="text-sm text-amber-500">
                    {t('apiKeys.copyWarning')}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded-md bg-muted px-3 py-2 text-sm font-mono break-all">
                    {newKeySecret}
                  </code>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => handleCopy(newKeySecret)}
                  >
                    {copied ? (
                      <Check className="size-4 text-emerald-500" />
                    ) : (
                      <Copy className="size-4" />
                    )}
                  </Button>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setNewKeySecret(null)
                    setShowCreate(false)
                  }}
                >
                  {t('common.done')}
                </Button>
              </div>
            ) : (
              <div className="flex gap-2">
                <Input
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder={t('apiKeys.keyNamePlaceholder')}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate()
                  }}
                />
                <Button onClick={handleCreate} disabled={creating || !newKeyName.trim()}>
                  {creating ? t('apiKeys.creating') : t('common.create')}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowCreate(false)
                    setNewKeyName("")
                  }}
                >
                  {t('common.cancel')}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Loading */}
      {loading && (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('apiKeys.loadingKeys')}</p>
        </div>
      )}

      {/* Active keys */}
      {!loading && activeKeys.length === 0 && !showCreate && (
        <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
          <KeyRound className="size-8 text-muted-foreground mb-2" />
          <p className="text-sm text-muted-foreground">{t('apiKeys.noKeys')}</p>
          <button
            onClick={() => setShowCreate(true)}
            className="text-sm text-primary hover:underline mt-1"
          >
            {t('apiKeys.generateFirst')}
          </button>
        </div>
      )}

      {!loading && activeKeys.length > 0 && (
        <div>
          <h2 className="text-sm font-semibold mb-2">{t('apiKeys.activeKeys')}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('apiKeys.name')}</TableHead>
                <TableHead>{t('apiKeys.prefix')}</TableHead>
                <TableHead>{t('apiKeys.created')}</TableHead>
                <TableHead>{t('apiKeys.lastUsed')}</TableHead>
                <TableHead>{t('apiKeys.expires')}</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {activeKeys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium">{key.name}</TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                      {key.prefix}...
                    </code>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(key.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {key.last_used
                      ? new Date(key.last_used).toLocaleDateString()
                      : t('common.never')}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {key.expires_at
                      ? new Date(key.expires_at).toLocaleDateString()
                      : t('common.never')}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => handleRevoke(key.id)}
                      disabled={revokingId === key.id}
                      title="Revoke key"
                    >
                      <Trash2 className="size-3 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Revoked keys */}
      {!loading && revokedKeys.length > 0 && (
        <div>
          <h2 className="text-sm font-semibold mb-2 text-muted-foreground">{t('apiKeys.revokedKeys')}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('apiKeys.name')}</TableHead>
                <TableHead>{t('apiKeys.prefix')}</TableHead>
                <TableHead>{t('apiKeys.created')}</TableHead>
                <TableHead>{t('invocations.status')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {revokedKeys.map((key) => (
                <TableRow key={key.id} className="opacity-50">
                  <TableCell className="font-medium">{key.name}</TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                      {key.prefix}...
                    </code>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(key.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Badge variant="destructive">{t('apiKeys.revoked')}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
