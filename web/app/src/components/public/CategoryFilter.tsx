import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { fetchCategories } from "@/api/client"
import type { Category } from "@/api/types"

interface CategoryFilterProps {
  selected: string | undefined
  onChange: (category: string | undefined) => void
}

export function CategoryFilter({ selected, onChange }: CategoryFilterProps) {
  const { t } = useTranslation()
  const [categories, setCategories] = useState<Category[]>([])

  useEffect(() => {
    fetchCategories()
      .then((res) => setCategories(res.categories ?? []))
      .catch(() => setCategories([]))
  }, [])

  if (categories.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2">
      <button
        onClick={() => onChange(undefined)}
        className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-all ${
          !selected
            ? "bg-primary text-primary-foreground shadow-[0_0_12px_oklch(0.72_0.15_192_/_0.15)]"
            : "bg-secondary/60 text-secondary-foreground hover:bg-secondary"
        }`}
      >
        {t('categoryFilter.all')}
      </button>
      {categories.map((cat) => (
        <button
          key={cat.id}
          onClick={() => onChange(cat.slug === selected ? undefined : cat.slug)}
          className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-all ${
            selected === cat.slug
              ? "bg-primary text-primary-foreground shadow-[0_0_12px_oklch(0.72_0.15_192_/_0.15)]"
              : "bg-secondary/60 text-secondary-foreground hover:bg-secondary"
          }`}
        >
          {cat.icon && <span className="mr-1">{cat.icon}</span>}
          {cat.name}
        </button>
      ))}
    </div>
  )
}
