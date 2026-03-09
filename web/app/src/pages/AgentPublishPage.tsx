import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderMutations } from "@/hooks/use-provider"
import { PublishWizard } from "@/components/provider/PublishWizard"
import type { PublishAgentData } from "@/hooks/use-provider"

export function AgentPublishPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { publishAgent } = useProviderMutations()

  const handleSubmit = async (data: PublishAgentData) => {
    const agent = await publishAgent(data)
    navigate(`/console/agents/${agent.id}`)
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">{t('provider.publishAgent')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('provider.registerAgent')}
        </p>
      </div>

      <PublishWizard onSubmit={handleSubmit} />
    </div>
  )
}
