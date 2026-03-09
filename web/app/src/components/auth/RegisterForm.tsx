import { useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"

interface RegisterFormProps {
  onSubmit: (
    email: string,
    password: string,
    displayName?: string
  ) => Promise<void>
  error?: string
}

export function RegisterForm({ onSubmit, error }: RegisterFormProps) {
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [loading, setLoading] = useState(false)
  const [localError, setLocalError] = useState("")
  const { t } = useTranslation()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError("")

    if (password !== confirmPassword) {
      setLocalError(t('auth.passwordMismatch'))
      return
    }
    if (password.length < 8) {
      setLocalError(t('auth.passwordMinLength'))
      return
    }

    setLoading(true)
    try {
      await onSubmit(email, password, displayName || undefined)
    } catch (err: any) {
      setLocalError(err.message || t('auth.registrationFailed'))
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
          htmlFor="display-name"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.displayName')}
        </label>
        <input
          id="display-name"
          type="text"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          autoComplete="name"
          className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder={t('auth.yourNameOptional')}
        />
      </div>

      <div>
        <label
          htmlFor="reg-email"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.email')}
        </label>
        <input
          id="reg-email"
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
          htmlFor="reg-password"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.password')}
        </label>
        <input
          id="reg-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="new-password"
          className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder={t('auth.atLeastChars')}
        />
      </div>

      <div>
        <label
          htmlFor="reg-confirm"
          className="block text-sm font-medium text-foreground mb-1"
        >
          {t('auth.confirmPassword')}
        </label>
        <input
          id="reg-confirm"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
          autoComplete="new-password"
          className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder={t('auth.confirmYourPassword')}
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {loading ? t('auth.creatingAccount') : t('auth.createAccount')}
      </button>

      <p className="text-center text-sm text-muted-foreground">
        {t('auth.alreadyHaveAccount')}{" "}
        <Link to="/login" className="text-primary hover:underline">
          {t('auth.signIn')}
        </Link>
      </p>
    </form>
  )
}
