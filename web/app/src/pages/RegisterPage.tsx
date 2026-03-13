import { useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { RegisterForm } from "@/components/auth/RegisterForm"
import { useTranslation } from "react-i18next"

export function RegisterPage() {
  const { register, pendingVerificationEmail } = useAuth()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const handleRegister = async (
    email: string,
    password: string,
    displayName?: string
  ) => {
    await register(email, password, displayName)
  }

  // Navigate to verify-email when pending verification is set.
  useEffect(() => {
    if (pendingVerificationEmail) {
      navigate("/verify-email", { replace: true })
    }
  }, [pendingVerificationEmail, navigate])

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <img src="/logo.jpg" alt="PeerClaw" className="mx-auto mb-4 size-12 rounded-md object-cover" />
        <h1 className="text-xl font-bold">{t('auth.createAnAccount')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t('auth.joinPlatform')}
        </p>
      </div>
      <RegisterForm onSubmit={handleRegister} />
    </div>
  )
}
