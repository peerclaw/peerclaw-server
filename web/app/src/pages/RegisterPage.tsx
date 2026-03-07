import { useNavigate } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { RegisterForm } from "@/components/auth/RegisterForm"

export function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()

  const handleRegister = async (
    email: string,
    password: string,
    displayName?: string
  ) => {
    await register(email, password, displayName)
    navigate("/", { replace: true })
  }

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <div className="mx-auto mb-4 size-10 rounded-md bg-primary flex items-center justify-center">
          <span className="text-sm font-bold text-primary-foreground">PC</span>
        </div>
        <h1 className="text-xl font-bold">Create an account</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Join the PeerClaw Agent Marketplace
        </p>
      </div>
      <RegisterForm onSubmit={handleRegister} />
    </div>
  )
}
