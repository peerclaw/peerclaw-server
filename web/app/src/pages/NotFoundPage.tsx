import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"

export function NotFoundPage() {
  const { t } = useTranslation()

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4">
      <h1 className="text-7xl font-bold text-foreground">404</h1>
      <p className="mt-4 text-lg text-muted-foreground">{t('notFound.description')}</p>
      <Button asChild className="mt-6">
        <Link to="/">{t('notFound.backHome')}</Link>
      </Button>
    </div>
  )
}
