import { useState, useEffect, useCallback, useRef } from "react"
import { useParams, useNavigate, useBlocker } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderAgent, useProviderMutations } from "@/hooks/use-provider"
import type { RegisterAgentData } from "@/hooks/use-provider"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Loader2, X } from "lucide-react"

const AUTH_TYPES = ["none", "api_key", "bearer_token", "oauth2"]
const PROTOCOL_OPTIONS = ["a2a", "mcp", "acp"]

export function AgentEditPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: agent, loading, error } = useProviderAgent(id)
  const { updateAgent } = useProviderMutations()

  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  // Form state — initialized from agent data once loaded
  const [form, setForm] = useState<Partial<RegisterAgentData> | null>(null)
  const [capInput, setCapInput] = useState("")
  const [tagInput, setTagInput] = useState("")

  const [isDirty, setIsDirty] = useState(false)
  const submittingRef = useRef(false)

  // Initialize form when agent data loads
  if (agent && !form) {
    setForm({
      name: agent.name,
      description: agent.description,
      version: agent.version,
      capabilities: agent.capabilities ?? [],
      protocols: agent.protocols ?? [],
      endpoint_url: agent.endpoint_url,
      auth_type: agent.auth_type ?? "none",
      tags: agent.tags ?? [],
      playground_enabled: agent.playground_enabled ?? false,
      visibility: agent.visibility ?? "public",
    })
  }

  // Track dirty state
  const markDirty = useCallback(() => setIsDirty(true), [])

  // Warn on browser close / tab close
  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener("beforeunload", handler)
    return () => window.removeEventListener("beforeunload", handler)
  }, [isDirty])

  // Block in-app navigation
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty && !submittingRef.current && currentLocation.pathname !== nextLocation.pathname
  )

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">{t('provider.loadingAgent')}</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-destructive">{error}</p>
      </div>
    )
  }

  if (!agent || !form) return null

  const update = (key: keyof RegisterAgentData, value: unknown) => {
    setForm((prev) => prev ? { ...prev, [key]: value } : prev)
    markDirty()
  }

  const toggleProtocol = (proto: string) => {
    const current = form.protocols ?? []
    update(
      "protocols",
      current.includes(proto) ? current.filter((p) => p !== proto) : [...current, proto]
    )
  }

  const addCap = () => {
    const trimmed = capInput.trim()
    if (trimmed && !(form.capabilities ?? []).includes(trimmed)) {
      update("capabilities", [...(form.capabilities ?? []), trimmed])
    }
    setCapInput("")
  }

  const addTag = () => {
    const trimmed = tagInput.trim()
    if (trimmed && !(form.tags ?? []).includes(trimmed)) {
      update("tags", [...(form.tags ?? []), trimmed])
    }
    setTagInput("")
  }

  const handleSubmit = async () => {
    if (!id || !form) return
    setSubmitting(true)
    setSubmitError(null)
    submittingRef.current = true
    try {
      await updateAgent(id, form as RegisterAgentData)
      setIsDirty(false)
      navigate(`/console/agents/${id}`)
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Failed to save")
      submittingRef.current = false
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">{t('provider.editAgent')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('provider.updateConfig', { name: agent.name })}
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('wizard.basicInfo')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Name */}
          <div>
            <label className="text-sm font-medium">{t('wizard.agentName')}</label>
            <Input
              value={form.name ?? ""}
              onChange={(e) => update("name", e.target.value)}
              className="mt-1"
            />
          </div>

          {/* Description */}
          <div>
            <label className="text-sm font-medium">{t('wizard.description')}</label>
            <textarea
              value={form.description ?? ""}
              onChange={(e) => update("description", e.target.value)}
              rows={3}
              className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
            />
          </div>

          {/* Version */}
          <div>
            <label className="text-sm font-medium">{t('wizard.version')}</label>
            <Input
              value={form.version ?? ""}
              onChange={(e) => update("version", e.target.value)}
              className="mt-1"
            />
          </div>

          {/* Tags */}
          <div>
            <label className="text-sm font-medium">{t('wizard.tags')}</label>
            <div className="flex gap-2 mt-1">
              <Input
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                placeholder={t('wizard.addTag')}
                onKeyDown={(e) => {
                  if (e.key === "Enter") { e.preventDefault(); addTag() }
                }}
              />
              <Button type="button" variant="outline" size="sm" onClick={addTag}>
                {t('common.add')}
              </Button>
            </div>
            <div className="flex flex-wrap gap-1.5 mt-2">
              {(form.tags ?? []).map((tag) => (
                <Badge key={tag} variant="secondary" className="gap-1">
                  {tag}
                  <button onClick={() => update("tags", (form.tags ?? []).filter((t) => t !== tag))}>
                    <X className="size-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('wizard.capabilities')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Capabilities */}
          <div>
            <div className="flex gap-2">
              <Input
                value={capInput}
                onChange={(e) => setCapInput(e.target.value)}
                placeholder={t('wizard.addCapability')}
                onKeyDown={(e) => {
                  if (e.key === "Enter") { e.preventDefault(); addCap() }
                }}
              />
              <Button type="button" variant="outline" size="sm" onClick={addCap}>
                {t('common.add')}
              </Button>
            </div>
            <div className="flex flex-wrap gap-1.5 mt-2">
              {(form.capabilities ?? []).map((cap) => (
                <Badge key={cap} variant="secondary" className="gap-1">
                  {cap}
                  <button onClick={() => update("capabilities", (form.capabilities ?? []).filter((c) => c !== cap))}>
                    <X className="size-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </div>

          {/* Protocols */}
          <div>
            <label className="text-sm font-medium">{t('wizard.protocols')}</label>
            <div className="flex flex-wrap gap-2 mt-2">
              {PROTOCOL_OPTIONS.map((proto) => (
                <button
                  key={proto}
                  type="button"
                  onClick={() => toggleProtocol(proto)}
                  className={`rounded-md border px-3 py-1.5 text-sm transition-colors ${
                    (form.protocols ?? []).includes(proto)
                      ? "border-primary bg-primary/10 text-primary font-medium"
                      : "border-border text-muted-foreground hover:border-foreground hover:text-foreground"
                  }`}
                >
                  {proto.toUpperCase()}
                </button>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('wizard.endpoint')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Endpoint URL */}
          <div>
            <label className="text-sm font-medium">{t('wizard.endpointUrl')}</label>
            <Input
              value={form.endpoint_url ?? ""}
              onChange={(e) => update("endpoint_url", e.target.value)}
              placeholder={t('wizard.endpointUrlPlaceholder')}
              className="mt-1"
            />
          </div>

          {/* Auth Type */}
          <div>
            <label className="text-sm font-medium">{t('wizard.authType')}</label>
            <select
              value={form.auth_type ?? "none"}
              onChange={(e) => update("auth_type", e.target.value)}
              className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            >
              {AUTH_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type === "none" ? t('wizard.none') : type === "api_key" ? t('wizard.apiKey') : type === "bearer_token" ? t('wizard.bearerToken') : t('wizard.oauth2')}
                </option>
              ))}
            </select>
          </div>

          {/* Playground toggle */}
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <label className="text-sm font-medium">{t('wizard.playgroundEnabled')}</label>
              <p className="text-xs text-muted-foreground">{t('wizard.playgroundEnabledDesc')}</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={form.playground_enabled ?? false}
              onClick={() => update("playground_enabled", !(form.playground_enabled ?? false))}
              className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                form.playground_enabled ? "bg-primary" : "bg-muted"
              }`}
            >
              <span
                className={`pointer-events-none inline-block size-4 rounded-full bg-background shadow-sm transition-transform ${
                  form.playground_enabled ? "translate-x-4" : "translate-x-0"
                }`}
              />
            </button>
          </div>

          {/* Visibility */}
          <div>
            <label className="text-sm font-medium">{t('wizard.visibility')}</label>
            <div className="flex gap-2 mt-2">
              {["public", "private"].map((v) => (
                <button
                  key={v}
                  type="button"
                  onClick={() => update("visibility", v)}
                  className={`rounded-md border px-3 py-1.5 text-sm transition-colors ${
                    (form.visibility ?? "public") === v
                      ? "border-primary bg-primary/10 text-primary font-medium"
                      : "border-border text-muted-foreground hover:border-foreground hover:text-foreground"
                  }`}
                >
                  {v === "public" ? t('wizard.visibilityPublic') : t('wizard.visibilityPrivate')}
                </button>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {submitError && (
        <p className="text-sm text-destructive">{submitError}</p>
      )}

      <div className="flex justify-end gap-3">
        <Button variant="outline" onClick={() => navigate(`/console/agents/${id}`)}>
          {t('common.cancel')}
        </Button>
        <Button onClick={handleSubmit} disabled={submitting}>
          {submitting && <Loader2 className="size-4 animate-spin" />}
          {submitting ? t('wizard.saving') : t('wizard.saveChanges')}
        </Button>
      </div>

      {/* Unsaved changes navigation blocker */}
      {blocker.state === "blocked" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="rounded-lg border border-border bg-card p-6 shadow-lg max-w-sm">
            <h3 className="text-lg font-semibold">{t('wizard.unsavedChangesTitle')}</h3>
            <p className="mt-2 text-sm text-muted-foreground">{t('wizard.unsavedChangesDesc')}</p>
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => blocker.reset?.()}>
                {t('common.cancel')}
              </Button>
              <Button variant="destructive" size="sm" onClick={() => blocker.proceed?.()}>
                {t('wizard.discardChanges')}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
