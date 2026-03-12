import { useState, useEffect } from "react"
import { Link, useNavigate } from "react-router-dom"
import { OTPInput } from "@/components/auth/OTPInput"
import { useTranslation } from "react-i18next"
import * as authAPI from "@/api/auth"

type Step = "email" | "reset"

export function ForgotPasswordPage() {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>("email")
  const [email, setEmail] = useState("")
  const [code, setCode] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [resendCooldown, setResendCooldown] = useState(0)

  useEffect(() => {
    if (resendCooldown <= 0) return
    const timer = setTimeout(() => setResendCooldown((c) => c - 1), 1000)
    return () => clearTimeout(timer)
  }, [resendCooldown])

  const handleSendCode = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (!email.includes("@")) {
      setError(t("auth.invalidEmail"))
      return
    }
    setLoading(true)
    try {
      await authAPI.forgotPassword(email)
      setStep("reset")
      setResendCooldown(60)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleResend = async () => {
    if (resendCooldown > 0) return
    try {
      await authAPI.forgotPassword(email)
      setResendCooldown(60)
    } catch {
      // ignore
    }
  }

  const handleReset = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (newPassword !== confirmPassword) {
      setError(t("auth.passwordMismatch"))
      return
    }
    if (newPassword.length < 8) {
      setError(t("auth.passwordMinLength"))
      return
    }
    if (code.length !== 6) {
      setError(t("auth.enterCode"))
      return
    }
    setLoading(true)
    try {
      await authAPI.resetPassword(email, code, newPassword)
      navigate("/login", { replace: true })
    } catch (err: any) {
      setError(err.message || t("auth.resetFailed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <img src="/logo.jpg" alt="PeerClaw" className="mx-auto mb-4 size-12 rounded-md object-cover" />
        <h1 className="text-xl font-bold">{t("auth.forgotPassword")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {step === "email"
            ? t("auth.forgotPasswordDesc")
            : t("auth.resetCodeSent", { email })}
        </p>
      </div>

      {error && (
        <div className="mb-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {step === "email" ? (
        <form onSubmit={handleSendCode} className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-foreground mb-1">
              {t("auth.email")}
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
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? t("auth.sending") : t("auth.sendResetCode")}
          </button>
          <p className="text-center text-sm text-muted-foreground">
            <Link to="/login" className="text-primary hover:underline">
              {t("auth.backToLogin")}
            </Link>
          </p>
        </form>
      ) : (
        <form onSubmit={handleReset} className="space-y-4">
          <OTPInput onComplete={setCode} disabled={loading} />

          <div>
            <label htmlFor="new-password" className="block text-sm font-medium text-foreground mb-1">
              {t("auth.newPassword")}
            </label>
            <input
              id="new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              autoComplete="new-password"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              placeholder={t("auth.atLeastChars")}
            />
          </div>

          <div>
            <label htmlFor="confirm-password" className="block text-sm font-medium text-foreground mb-1">
              {t("auth.confirmPassword")}
            </label>
            <input
              id="confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              autoComplete="new-password"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              placeholder={t("auth.confirmYourPassword")}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? t("auth.resetting") : t("auth.resetPassword")}
          </button>

          <div className="text-center">
            <button
              type="button"
              onClick={handleResend}
              disabled={resendCooldown > 0}
              className="text-sm text-primary hover:underline disabled:opacity-50 disabled:no-underline"
            >
              {resendCooldown > 0
                ? t("auth.resendIn", { seconds: resendCooldown })
                : t("auth.resendCode")}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}
