import { useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"

interface LoginFormProps {
  onSubmit: (email: string, password: string) => Promise<void>
  error?: string
}

export function LoginForm({ onSubmit, error }: LoginFormProps) {
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [localError, setLocalError] = useState("")
  const { t } = useTranslation()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError("")
    setLoading(true)
    try {
      await onSubmit(email, password)
    } catch (err: any) {
      setLocalError(err.message || t('auth.loginFailed'))
    } finally {
      setLoading(false)
    }
  }

  const displayError = error || localError

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {displayError && (
        <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {displayError}
        </div>
      )}

      <div>
        <label
          htmlFor="email"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.email')}
        </label>
        <input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="email"
          className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder="you@example.com"
        />
      </div>

      <div>
        <label
          htmlFor="password"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.password')}
        </label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="current-password"
          className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder={t('auth.enterPassword')}
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {loading ? t('auth.signingIn') : t('nav.signIn')}
      </button>

      <div className="flex items-center justify-between text-sm">
        <Link to="/forgot-password" className="text-primary hover:underline">
          {t('auth.forgotPassword')}
        </Link>
      </div>

      <p className="text-center text-sm text-muted-foreground">
        {t('auth.dontHaveAccount')}{" "}
        <Link to="/register" className="text-primary hover:underline">
          {t('auth.signUp')}
        </Link>
      </p>
    </form>
  )
}
