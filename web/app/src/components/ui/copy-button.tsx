import { useState, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Copy, Check } from "lucide-react"
import { useTranslation } from "react-i18next"
import { cn } from "@/lib/utils"

interface CopyButtonProps {
  value: string
  size?: "sm" | "default" | "icon"
  variant?: "ghost" | "outline" | "default"
  className?: string
}

export function CopyButton({ value, size = "icon", variant = "ghost", className }: CopyButtonProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement("textarea")
      textarea.value = value
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand("copy")
      document.body.removeChild(textarea)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [value])

  return (
    <Button
      variant={variant}
      size={size}
      onClick={handleCopy}
      className={cn("shrink-0", className)}
      title={t('common.copyToClipboard')}
    >
      {copied ? (
        <Check className="size-3.5 text-green-500" />
      ) : (
        <Copy className="size-3.5" />
      )}
    </Button>
  )
}
