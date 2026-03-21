import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Check, X } from "lucide-react"

interface PasswordStrengthProps {
  password: string
}

interface Requirement {
  key: string
  met: boolean
}

function getScore(password: string): number {
  let score = 0
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (/[A-Z]/.test(password) && /[a-z]/.test(password)) score++
  if (/\d/.test(password)) score++
  if (/[^A-Za-z0-9]/.test(password)) score++
  return Math.min(score, 4)
}

const COLORS = [
  "bg-destructive",
  "bg-orange-500",
  "bg-yellow-500",
  "bg-emerald-500",
]

export function PasswordStrength({ password }: PasswordStrengthProps) {
  const { t } = useTranslation()

  const score = useMemo(() => getScore(password), [password])

  const requirements: Requirement[] = useMemo(
    () => [
      { key: "auth.minLength", met: password.length >= 8 },
      { key: "auth.hasUppercase", met: /[A-Z]/.test(password) },
      { key: "auth.hasLowercase", met: /[a-z]/.test(password) },
      { key: "auth.hasNumber", met: /\d/.test(password) },
      { key: "auth.hasSpecial", met: /[^A-Za-z0-9]/.test(password) },
    ],
    [password]
  )

  const labels = ["auth.weak", "auth.fair", "auth.good", "auth.strong"]
  const label = score > 0 ? t(labels[score - 1]) : ""

  if (!password) return null

  return (
    <div className="space-y-2 mt-2">
      <div className="flex items-center gap-2">
        <div className="flex flex-1 gap-1">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className={`h-1.5 flex-1 rounded-full transition-colors ${
                i < score ? COLORS[score - 1] : "bg-muted"
              }`}
            />
          ))}
        </div>
        {label && (
          <span className="text-xs text-muted-foreground">{label}</span>
        )}
      </div>
      <ul className="space-y-1">
        {requirements.map((req) => (
          <li
            key={req.key}
            className={`flex items-center gap-1.5 text-xs ${
              req.met ? "text-emerald-500" : "text-muted-foreground"
            }`}
          >
            {req.met ? (
              <Check className="size-3" />
            ) : (
              <X className="size-3" />
            )}
            {t(req.key)}
          </li>
        ))}
      </ul>
    </div>
  )
}
