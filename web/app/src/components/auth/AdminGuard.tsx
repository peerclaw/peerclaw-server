import { Navigate, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth"

export function AdminGuard({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground text-sm">
        Loading...
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (user.role !== "admin") {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}
