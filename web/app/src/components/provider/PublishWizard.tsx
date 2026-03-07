import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ChevronLeft, ChevronRight, Loader2, X } from "lucide-react"
import type { PublishAgentData } from "@/hooks/use-provider"

interface PublishWizardProps {
  onSubmit: (data: PublishAgentData) => Promise<void>
  initialData?: Partial<PublishAgentData>
}

const STEPS = [
  { title: "Basic Info", description: "Name and description for your agent" },
  { title: "Capabilities", description: "What your agent can do" },
  { title: "Endpoint", description: "Where your agent is hosted" },
  { title: "Authentication", description: "How callers authenticate" },
  { title: "Preview", description: "Review and publish" },
]

const AUTH_TYPES = ["none", "api_key", "bearer_token", "oauth2"]
const PROTOCOL_OPTIONS = ["a2a", "mcp", "http", "grpc"]

export function PublishWizard({ onSubmit, initialData }: PublishWizardProps) {
  const [step, setStep] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Form state
  const [name, setName] = useState(initialData?.name ?? "")
  const [description, setDescription] = useState(initialData?.description ?? "")
  const [version, setVersion] = useState(initialData?.version ?? "1.0.0")
  const [capabilities, setCapabilities] = useState<string[]>(initialData?.capabilities ?? [])
  const [capabilityInput, setCapabilityInput] = useState("")
  const [protocols, setProtocols] = useState<string[]>(initialData?.protocols ?? [])
  const [tags, setTags] = useState<string[]>(initialData?.tags ?? [])
  const [tagInput, setTagInput] = useState("")
  const [endpointUrl, setEndpointUrl] = useState(initialData?.endpoint_url ?? "")
  const [authType, setAuthType] = useState(initialData?.auth_type ?? "none")
  const [authHeader, setAuthHeader] = useState(initialData?.auth_config?.header ?? "")

  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  // Capability management
  const addCapability = () => {
    const trimmed = capabilityInput.trim()
    if (trimmed && !capabilities.includes(trimmed)) {
      setCapabilities([...capabilities, trimmed])
    }
    setCapabilityInput("")
  }

  const removeCapability = (cap: string) => {
    setCapabilities(capabilities.filter((c) => c !== cap))
  }

  // Tag management
  const addTag = () => {
    const trimmed = tagInput.trim()
    if (trimmed && !tags.includes(trimmed)) {
      setTags([...tags, trimmed])
    }
    setTagInput("")
  }

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag))
  }

  // Protocol toggle
  const toggleProtocol = (proto: string) => {
    setProtocols((prev) =>
      prev.includes(proto) ? prev.filter((p) => p !== proto) : [...prev, proto]
    )
  }

  // Validation per step
  const validateStep = (s: number): boolean => {
    const errors: Record<string, string> = {}

    switch (s) {
      case 0:
        if (!name.trim()) errors.name = "Name is required"
        if (!description.trim()) errors.description = "Description is required"
        if (!version.trim()) errors.version = "Version is required"
        break
      case 1:
        if (capabilities.length === 0) errors.capabilities = "Add at least one capability"
        if (protocols.length === 0) errors.protocols = "Select at least one protocol"
        break
      case 2:
        if (!endpointUrl.trim()) errors.endpointUrl = "Endpoint URL is required"
        try {
          new URL(endpointUrl)
        } catch {
          if (endpointUrl.trim()) errors.endpointUrl = "Must be a valid URL"
        }
        break
      case 3:
        // Auth step is optional, no hard requirements
        break
    }

    setValidationErrors(errors)
    return Object.keys(errors).length === 0
  }

  const goNext = () => {
    if (validateStep(step)) {
      setStep((s) => Math.min(s + 1, STEPS.length - 1))
    }
  }

  const goBack = () => {
    setValidationErrors({})
    setStep((s) => Math.max(s - 1, 0))
  }

  const handleSubmit = async () => {
    const data: PublishAgentData = {
      name: name.trim(),
      description: description.trim(),
      version: version.trim(),
      capabilities,
      protocols,
      endpoint_url: endpointUrl.trim(),
      auth_type: authType,
      tags,
    }

    if (authType !== "none" && authHeader.trim()) {
      data.auth_config = { header: authHeader.trim() }
    }

    setSubmitting(true)
    setError(null)
    try {
      await onSubmit(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to publish agent")
    } finally {
      setSubmitting(false)
    }
  }

  const renderFieldError = (key: string) => {
    if (!validationErrors[key]) return null
    return <p className="text-xs text-destructive mt-1">{validationErrors[key]}</p>
  }

  const renderStep = () => {
    switch (step) {
      case 0:
        return (
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-foreground">Agent Name</label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-awesome-agent"
                className="mt-1"
              />
              {renderFieldError("name")}
            </div>
            <div>
              <label className="text-sm font-medium text-foreground">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe what your agent does..."
                rows={3}
                className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
              />
              {renderFieldError("description")}
            </div>
            <div>
              <label className="text-sm font-medium text-foreground">Version</label>
              <Input
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                placeholder="1.0.0"
                className="mt-1"
              />
              {renderFieldError("version")}
            </div>
            <div>
              <label className="text-sm font-medium text-foreground">Tags</label>
              <div className="flex gap-2 mt-1">
                <Input
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  placeholder="Add a tag..."
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault()
                      addTag()
                    }
                  }}
                />
                <Button type="button" variant="outline" size="sm" onClick={addTag}>
                  Add
                </Button>
              </div>
              <div className="flex flex-wrap gap-1.5 mt-2">
                {tags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="gap-1">
                    {tag}
                    <button onClick={() => removeTag(tag)}>
                      <X className="size-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          </div>
        )

      case 1:
        return (
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-foreground">Capabilities</label>
              <div className="flex gap-2 mt-1">
                <Input
                  value={capabilityInput}
                  onChange={(e) => setCapabilityInput(e.target.value)}
                  placeholder="e.g. code-review, summarize, translate"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault()
                      addCapability()
                    }
                  }}
                />
                <Button type="button" variant="outline" size="sm" onClick={addCapability}>
                  Add
                </Button>
              </div>
              {renderFieldError("capabilities")}
              <div className="flex flex-wrap gap-1.5 mt-2">
                {capabilities.map((cap) => (
                  <Badge key={cap} variant="secondary" className="gap-1">
                    {cap}
                    <button onClick={() => removeCapability(cap)}>
                      <X className="size-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
            <div>
              <label className="text-sm font-medium text-foreground">Protocols</label>
              {renderFieldError("protocols")}
              <div className="flex flex-wrap gap-2 mt-2">
                {PROTOCOL_OPTIONS.map((proto) => (
                  <button
                    key={proto}
                    type="button"
                    onClick={() => toggleProtocol(proto)}
                    className={`rounded-md border px-3 py-1.5 text-sm transition-colors ${
                      protocols.includes(proto)
                        ? "border-primary bg-primary/10 text-primary font-medium"
                        : "border-border text-muted-foreground hover:border-foreground hover:text-foreground"
                    }`}
                  >
                    {proto.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
          </div>
        )

      case 2:
        return (
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-foreground">Endpoint URL</label>
              <Input
                value={endpointUrl}
                onChange={(e) => setEndpointUrl(e.target.value)}
                placeholder="https://api.example.com/agent"
                className="mt-1"
              />
              {renderFieldError("endpointUrl")}
              <p className="text-xs text-muted-foreground mt-1">
                The URL where your agent receives invocation requests.
              </p>
            </div>
          </div>
        )

      case 3:
        return (
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-foreground">Authentication Type</label>
              <div className="flex flex-wrap gap-2 mt-2">
                {AUTH_TYPES.map((type) => (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setAuthType(type)}
                    className={`rounded-md border px-3 py-1.5 text-sm transition-colors ${
                      authType === type
                        ? "border-primary bg-primary/10 text-primary font-medium"
                        : "border-border text-muted-foreground hover:border-foreground hover:text-foreground"
                    }`}
                  >
                    {type === "none"
                      ? "None"
                      : type === "api_key"
                        ? "API Key"
                        : type === "bearer_token"
                          ? "Bearer Token"
                          : "OAuth2"}
                  </button>
                ))}
              </div>
            </div>
            {authType !== "none" && (
              <div>
                <label className="text-sm font-medium text-foreground">
                  Auth Header Name
                </label>
                <Input
                  value={authHeader}
                  onChange={(e) => setAuthHeader(e.target.value)}
                  placeholder="X-API-Key"
                  className="mt-1"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  The HTTP header callers should use for authentication.
                </p>
              </div>
            )}
          </div>
        )

      case 4:
        return (
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-foreground">Review Your Agent</h3>
            <div className="grid gap-3 text-sm">
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Name</span>
                <span className="font-medium">{name}</span>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Version</span>
                <span className="font-medium">{version}</span>
              </div>
              <div className="border-b border-border pb-2">
                <span className="text-muted-foreground">Description</span>
                <p className="mt-1 text-foreground">{description}</p>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Capabilities</span>
                <div className="flex flex-wrap gap-1 justify-end">
                  {capabilities.map((c) => (
                    <Badge key={c} variant="secondary">
                      {c}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Protocols</span>
                <div className="flex gap-1">
                  {protocols.map((p) => (
                    <Badge key={p} variant="outline">
                      {p.toUpperCase()}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Endpoint</span>
                <span className="font-mono text-xs">{endpointUrl}</span>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-muted-foreground">Auth Type</span>
                <span className="font-medium">{authType}</span>
              </div>
              {tags.length > 0 && (
                <div className="flex justify-between pb-2">
                  <span className="text-muted-foreground">Tags</span>
                  <div className="flex flex-wrap gap-1 justify-end">
                    {tags.map((t) => (
                      <Badge key={t} variant="secondary">
                        {t}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )

      default:
        return null
    }
  }

  return (
    <Card>
      <CardHeader>
        {/* Step indicator */}
        <div className="flex items-center gap-2 mb-2">
          {STEPS.map((_s, i) => (
            <div key={i} className="flex items-center gap-2">
              <div
                className={`flex size-7 items-center justify-center rounded-full text-xs font-medium ${
                  i === step
                    ? "bg-primary text-primary-foreground"
                    : i < step
                      ? "bg-primary/20 text-primary"
                      : "bg-muted text-muted-foreground"
                }`}
              >
                {i + 1}
              </div>
              {i < STEPS.length - 1 && (
                <div
                  className={`h-px w-6 ${
                    i < step ? "bg-primary" : "bg-border"
                  }`}
                />
              )}
            </div>
          ))}
        </div>
        <CardTitle>{STEPS[step].title}</CardTitle>
        <p className="text-sm text-muted-foreground">{STEPS[step].description}</p>
      </CardHeader>

      <CardContent>
        {renderStep()}

        {error && (
          <p className="text-sm text-destructive mt-4">{error}</p>
        )}

        <div className="flex items-center justify-between mt-6 pt-4 border-t border-border">
          <Button
            variant="outline"
            onClick={goBack}
            disabled={step === 0}
          >
            <ChevronLeft className="size-4" />
            Back
          </Button>

          {step < STEPS.length - 1 ? (
            <Button onClick={goNext}>
              Next
              <ChevronRight className="size-4" />
            </Button>
          ) : (
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting && <Loader2 className="size-4 animate-spin" />}
              {submitting ? "Publishing..." : "Publish Agent"}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
