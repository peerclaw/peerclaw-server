import { useState } from "react"
import { useAuth } from "@/hooks/use-auth"
import { useTranslation } from "react-i18next"

export function ProfilePage() {
  const { user, updateProfile, changePassword } = useAuth()
  const { t } = useTranslation()

  const [displayName, setDisplayName] = useState(user?.display_name || "")
  const [email, setEmail] = useState(user?.email || "")
  const [description, setDescription] = useState(user?.description || "")
  const [profileLoading, setProfileLoading] = useState(false)
  const [profileMsg, setProfileMsg] = useState("")
  const [profileError, setProfileError] = useState("")

  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [passwordMsg, setPasswordMsg] = useState("")
  const [passwordError, setPasswordError] = useState("")

  const handleProfileSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setProfileMsg("")
    setProfileError("")
    setProfileLoading(true)
    try {
      await updateProfile({
        display_name: displayName,
        email,
        description,
      })
      setProfileMsg(t("profilePage.profileUpdated"))
    } catch (err: any) {
      setProfileError(err.message || t("common.error"))
    } finally {
      setProfileLoading(false)
    }
  }

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setPasswordMsg("")
    setPasswordError("")

    if (newPassword !== confirmPassword) {
      setPasswordError(t("profilePage.passwordMismatch"))
      return
    }
    if (newPassword.length < 8) {
      setPasswordError(t("profilePage.passwordMinLength"))
      return
    }

    setPasswordLoading(true)
    try {
      await changePassword(currentPassword, newPassword)
      setPasswordMsg(t("profilePage.passwordChanged"))
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err: any) {
      setPasswordError(err.message || t("common.error"))
    } finally {
      setPasswordLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-8">
      <div>
        <h1 className="text-2xl font-bold">{t("profilePage.title")}</h1>
      </div>

      {/* Profile section */}
      <div className="rounded-lg border border-border bg-card p-6">
        <h2 className="text-lg font-semibold mb-4">{t("profilePage.profileSection")}</h2>
        {profileMsg && (
          <div className="mb-4 rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-500">
            {profileMsg}
          </div>
        )}
        {profileError && (
          <div className="mb-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {profileError}
          </div>
        )}
        <form onSubmit={handleProfileSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.displayName")}
            </label>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.email")}
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.description")}
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder={t("profilePage.descriptionPlaceholder")}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            />
          </div>
          <button
            type="submit"
            disabled={profileLoading}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {profileLoading ? t("profilePage.saving") : t("profilePage.saveProfile")}
          </button>
        </form>
      </div>

      {/* Password section */}
      <div className="rounded-lg border border-border bg-card p-6">
        <h2 className="text-lg font-semibold mb-4">{t("profilePage.passwordSection")}</h2>
        {passwordMsg && (
          <div className="mb-4 rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-500">
            {passwordMsg}
          </div>
        )}
        {passwordError && (
          <div className="mb-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {passwordError}
          </div>
        )}
        <form onSubmit={handlePasswordSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.currentPassword")}
            </label>
            <input
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.newPassword")}
            </label>
            <input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              {t("profilePage.confirmNewPassword")}
            </label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <button
            type="submit"
            disabled={passwordLoading}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {passwordLoading ? t("profilePage.changing") : t("profilePage.changePassword")}
          </button>
        </form>
      </div>
    </div>
  )
}
