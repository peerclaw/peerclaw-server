import { useRef, useState, useCallback } from "react"

interface OTPInputProps {
  length?: number
  onComplete: (code: string) => void
  disabled?: boolean
}

export function OTPInput({ length = 6, onComplete, disabled }: OTPInputProps) {
  const [values, setValues] = useState<string[]>(Array(length).fill(""))
  const inputs = useRef<(HTMLInputElement | null)[]>([])

  const focusInput = useCallback((index: number) => {
    if (index >= 0 && index < length) {
      inputs.current[index]?.focus()
    }
  }, [length])

  const handleChange = useCallback(
    (index: number, value: string) => {
      // Only accept digits.
      const digit = value.replace(/\D/g, "").slice(-1)
      const newValues = [...values]
      newValues[index] = digit
      setValues(newValues)

      if (digit && index < length - 1) {
        focusInput(index + 1)
      }

      const code = newValues.join("")
      if (code.length === length && newValues.every((v) => v !== "")) {
        onComplete(code)
      }
    },
    [values, length, onComplete, focusInput]
  )

  const handleKeyDown = useCallback(
    (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Backspace" && !values[index] && index > 0) {
        focusInput(index - 1)
      }
    },
    [values, focusInput]
  )

  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      e.preventDefault()
      const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, length)
      if (!pasted) return
      const newValues = [...values]
      for (let i = 0; i < pasted.length; i++) {
        newValues[i] = pasted[i]
      }
      setValues(newValues)
      focusInput(Math.min(pasted.length, length - 1))

      if (pasted.length === length) {
        onComplete(pasted)
      }
    },
    [values, length, onComplete, focusInput]
  )

  return (
    <div className="flex justify-center gap-2" onPaste={handlePaste}>
      {values.map((val, i) => (
        <input
          key={i}
          ref={(el) => { inputs.current[i] = el }}
          type="text"
          inputMode="numeric"
          maxLength={1}
          value={val}
          disabled={disabled}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          onFocus={(e) => e.target.select()}
          className="size-12 rounded-lg border border-input bg-background text-center text-lg font-mono focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
        />
      ))}
    </div>
  )
}
