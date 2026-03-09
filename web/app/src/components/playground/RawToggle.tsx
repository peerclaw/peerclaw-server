import { useTranslation } from "react-i18next"
import { Code } from "lucide-react"

interface RawToggleProps {
  data: unknown
  show: boolean
  onToggle: () => void
}

export function RawToggle({ data, show, onToggle }: RawToggleProps) {
  const { t } = useTranslation()

  if (!data) return null

  return (
    <div className="border-t border-border">
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-2 px-4 py-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
      >
        <Code className="size-3.5" />
        {show ? t('playground.hideRaw') : t('playground.showRaw')}
      </button>

      {show && (
        <div className="max-h-64 overflow-auto border-t border-border bg-muted/30 px-4 py-3">
          <pre className="text-xs leading-relaxed text-muted-foreground">
            <code>
              {typeof data === "string"
                ? data
                : JSON.stringify(data, null, 2)}
            </code>
          </pre>
        </div>
      )}
    </div>
  )
}
