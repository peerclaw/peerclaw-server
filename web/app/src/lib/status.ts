export type BadgeVariant = "default" | "secondary" | "destructive" | "outline"

export function agentStatusVariant(
  status: string
): BadgeVariant {
  switch (status) {
    case "online":
      return "default"
    case "degraded":
      return "secondary"
    case "offline":
      return "outline"
    default:
      return "outline"
  }
}

export function reportStatusVariant(
  status: string
): BadgeVariant {
  switch (status) {
    case "pending":
      return "secondary"
    case "reviewed":
      return "default"
    case "dismissed":
      return "outline"
    case "actioned":
      return "destructive"
    default:
      return "outline"
  }
}

export function httpStatusVariant(code: number): BadgeVariant {
  if (code >= 200 && code < 300) return "default"
  if (code >= 400 && code < 500) return "secondary"
  return "destructive"
}

export function roleVariant(role: string): BadgeVariant {
  switch (role) {
    case "admin":
      return "destructive"
    case "provider":
      return "secondary"
    default:
      return "default"
  }
}
