import { useEffect } from "react"
import { useNavigate, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { LoginForm } from "@/components/auth/LoginForm"
import { useTranslation } from "react-i18next"

export function LoginPage() {
  const { login, pendingVerificationEmail } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as any)?.from?.pathname || "/"
  const { t } = useTranslation()

  useEffect(() => {
    if (pendingVerificationEmail) {
      navigate("/verify-email", { replace: true })
    }
  }, [pendingVerificationEmail, navigate])

  const handleLogin = async (email: string, password: string) => {
    await login(email, password)
    navigate(from, { replace: true })
  }

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <img src="/logo.jpg" alt="PeerClaw" className="mx-auto mb-4 size-12 rounded-md object-cover" />
        <h1 className="text-xl font-bold">{t('auth.signInToPeerclaw')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t('auth.accessMarketplace')}
        </p>
      </div>
      <LoginForm onSubmit={handleLogin} />
    </div>
  )
}
