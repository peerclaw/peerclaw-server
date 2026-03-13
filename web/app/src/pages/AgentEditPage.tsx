import { useParams, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderAgent, useProviderMutations } from "@/hooks/use-provider"
import { RegisterWizard } from "@/components/provider/RegisterWizard"
import type { RegisterAgentData } from "@/hooks/use-provider"

export function AgentEditPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: agent, loading, error } = useProviderAgent(id)
  const { updateAgent } = useProviderMutations()

  const handleSubmit = async (data: RegisterAgentData) => {
    if (!id) return
    await updateAgent(id, data)
    navigate(`/console/agents/${id}`)
  }

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

  if (!agent) return null

  const initialData: Partial<RegisterAgentData> = {
    name: agent.name,
    description: agent.description,
    version: agent.version,
    capabilities: agent.capabilities,
    protocols: agent.protocols,
    endpoint_url: agent.endpoint_url,
    auth_type: agent.auth_type,
    tags: agent.tags,
    playground_enabled: agent.playground_enabled,
    visibility: agent.visibility,
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">{t('provider.editAgent')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('provider.updateConfig', { name: agent.name })}
        </p>
      </div>

      <RegisterWizard onSubmit={handleSubmit} initialData={initialData} editMode />
    </div>
  )
}
