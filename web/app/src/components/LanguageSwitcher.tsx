import { useTranslation } from "react-i18next"

export function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const isZh = i18n.language.startsWith("zh")

  const toggle = () => {
    i18n.changeLanguage(isZh ? "en" : "zh")
  }

  return (
    <button
      onClick={toggle}
      className="rounded-md border border-border px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
    >
      {isZh ? "EN" : "中文"}
    </button>
  )
}
