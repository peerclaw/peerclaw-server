import { useEffect, useState } from "react"
import { fetchCategories } from "@/api/client"
import type { Category } from "@/api/types"

interface CategoryFilterProps {
  selected: string | undefined
  onChange: (category: string | undefined) => void
}

export function CategoryFilter({ selected, onChange }: CategoryFilterProps) {
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
        className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
          !selected
            ? "bg-primary text-primary-foreground"
            : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
        }`}
      >
        All
      </button>
      {categories.map((cat) => (
        <button
          key={cat.id}
          onClick={() => onChange(cat.slug === selected ? undefined : cat.slug)}
          className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            selected === cat.slug
              ? "bg-primary text-primary-foreground"
              : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
          }`}
        >
          {cat.icon && <span className="mr-1">{cat.icon}</span>}
          {cat.name}
        </button>
      ))}
    </div>
  )
}
