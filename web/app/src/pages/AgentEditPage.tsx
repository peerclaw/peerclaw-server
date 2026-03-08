import { useParams, useNavigate } from "react-router-dom"
import { useProviderAgent, useProviderMutations } from "@/hooks/use-provider"
import { PublishWizard } from "@/components/provider/PublishWizard"
import type { PublishAgentData } from "@/hooks/use-provider"

export function AgentEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: agent, loading, error } = useProviderAgent(id)
  const { updateAgent } = useProviderMutations()

  const handleSubmit = async (data: PublishAgentData) => {
    if (!id) return
    await updateAgent(id, data)
    navigate(`/console/agents/${id}`)
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading agent...</p>
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

  const initialData: Partial<PublishAgentData> = {
    name: agent.name,
    description: agent.description,
    version: agent.version,
    capabilities: agent.capabilities,
    protocols: agent.protocols,
    endpoint_url: agent.endpoint_url,
    auth_type: agent.auth_type,
    tags: agent.tags,
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">Edit Agent</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Update {agent.name} configuration
        </p>
      </div>

      <PublishWizard onSubmit={handleSubmit} initialData={initialData} editMode />
    </div>
  )
}
