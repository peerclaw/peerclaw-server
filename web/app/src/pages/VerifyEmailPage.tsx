import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"
import { OTPInput } from "@/components/auth/OTPInput"
import { useTranslation } from "react-i18next"

export function VerifyEmailPage() {
  const { pendingVerificationEmail, verifyEmail, resendVerification, clearPendingVerification } = useAuth()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [resendCooldown, setResendCooldown] = useState(0)

  const email = pendingVerificationEmail

  useEffect(() => {
    if (!email) {
      navigate("/register", { replace: true })
    }
    return () => { clearPendingVerification() }
  }, [email, navigate, clearPendingVerification])

  useEffect(() => {
    if (resendCooldown <= 0) return
    const timer = setTimeout(() => setResendCooldown((c) => c - 1), 1000)
    return () => clearTimeout(timer)
  }, [resendCooldown])

  const handleVerify = async (code: string) => {
    if (!email) return
    setError("")
    setLoading(true)
    try {
      await verifyEmail(email, code)
      navigate("/", { replace: true })
    } catch (err: any) {
      setError(err.message || t("auth.verificationFailed"))
    } finally {
      setLoading(false)
    }
  }

  const handleResend = async () => {
    if (!email || resendCooldown > 0) return
    try {
      await resendVerification(email)
      setResendCooldown(60)
    } catch (err: any) {
      setError(err.message || t("auth.resendFailed"))
    }
  }

  if (!email) return null

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <div className="mb-8 text-center">
        <img src="/logo.jpg" alt="PeerClaw" className="mx-auto mb-4 size-12 rounded-md object-cover" />
        <h1 className="text-xl font-bold">{t("auth.verifyYourEmail")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("auth.verificationSent", { email })}
        </p>
      </div>

      <div className="space-y-6">
        {error && (
          <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        )}

        <OTPInput onComplete={handleVerify} disabled={loading} />

        {loading && (
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.verifying")}
          </p>
        )}

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
      </div>
    </div>
  )
}
