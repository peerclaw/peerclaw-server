import { TableHead } from "@/components/ui/table"
import { cn } from "@/lib/utils"

interface SortableHeaderProps {
  field: string
  label: string
  currentSort: string
  onSort: (sort: string) => void
}

function SortableHeader({ field, label, currentSort, onSort }: SortableHeaderProps) {
  const isAsc = currentSort === field
  const isDesc = currentSort === `-${field}`

  const handleClick = () => {
    if (!isAsc && !isDesc) {
      onSort(field)
    } else if (isAsc) {
      onSort(`-${field}`)
    } else {
      onSort("")
    }
  }

  return (
    <TableHead
      className="cursor-pointer select-none"
      onClick={handleClick}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        <span
          className={cn(
            "text-xs",
            !isAsc && !isDesc && "text-muted-foreground/40"
          )}
        >
          {isAsc ? "\u25B2" : isDesc ? "\u25BC" : "\u21C5"}
        </span>
      </span>
    </TableHead>
  )
}

export { SortableHeader }
