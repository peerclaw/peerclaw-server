import { useTranslation } from "react-i18next"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Globe, Check } from "lucide-react"

const LANGUAGES = [
  { code: "en", native: "English" },
  { code: "zh", native: "中文" },
  { code: "es", native: "Español" },
  { code: "fr", native: "Français" },
  { code: "ar", native: "العربية" },
  { code: "pt", native: "Português" },
  { code: "ja", native: "日本語" },
  { code: "ru", native: "Русский" },
]

export function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const currentLang = i18n.language

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex items-center gap-1.5 rounded-md border border-border px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground">
          <Globe className="size-3.5" />
          {LANGUAGES.find((l) => currentLang.startsWith(l.code))?.native || "English"}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-[140px]">
        {LANGUAGES.map((lang) => (
          <DropdownMenuItem
            key={lang.code}
            onClick={() => i18n.changeLanguage(lang.code)}
          >
            {currentLang.startsWith(lang.code) && <Check className="size-3.5" />}
            <span className={currentLang.startsWith(lang.code) ? "" : "pl-5.5"}>{lang.native}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
