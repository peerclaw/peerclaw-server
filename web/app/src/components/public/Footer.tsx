import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"

export function Footer() {
  const { t } = useTranslation()

  return (
    <footer className="border-t border-border bg-card">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
        <p className="text-xs text-muted-foreground">
          {t('footer.copyright', { year: new Date().getFullYear() })}
        </p>
        <nav className="flex items-center gap-4">
          <Link to="/directory" className="text-xs text-muted-foreground hover:text-foreground transition-colors">
            {t('nav.directory')}
          </Link>
          <Link to="/about" className="text-xs text-muted-foreground hover:text-foreground transition-colors">
            {t('nav.about')}
          </Link>
          <a
            href="https://github.com/peerclaw/peerclaw"
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            GitHub
          </a>
          <LanguageSwitcher />
        </nav>
      </div>
    </footer>
  )
}
