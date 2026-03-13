import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderMutations } from "@/hooks/use-provider"
import { RegisterWizard } from "@/components/provider/RegisterWizard"
import type { RegisterAgentData } from "@/hooks/use-provider"

export function AgentRegisterPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { registerAgent } = useProviderMutations()

  const handleSubmit = async (data: RegisterAgentData) => {
    const agent = await registerAgent(data)
    navigate(`/console/agents/${agent.id}`)
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">{t('nav.registerAgent')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('provider.registerAgent')}
        </p>
      </div>

      <RegisterWizard onSubmit={handleSubmit} />
    </div>
  )
}
