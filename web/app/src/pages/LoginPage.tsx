import { useNavigate, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { LoginForm } from "@/components/auth/LoginForm"

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as any)?.from?.pathname || "/"

  const handleLogin = async (email: string, password: string) => {
    await login(email, password)
    navigate(from, { replace: true })
  }

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <div className="mx-auto mb-4 size-10 rounded-md bg-primary flex items-center justify-center">
          <span className="text-sm font-bold text-primary-foreground">PC</span>
        </div>
        <h1 className="text-xl font-bold">Sign in to PeerClaw</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Access the Agent Marketplace
        </p>
      </div>
      <LoginForm onSubmit={handleLogin} />
    </div>
  )
}
