import { useState, useCallback } from "react"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { useTranslation } from "react-i18next"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

export interface BulkAction {
  label: string
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost"
  onClick: (ids: string[]) => void
}

interface SelectableTableProps<T> {
  items: T[]
  getKey: (item: T) => string
  columns: React.ReactNode
  renderRow: (item: T, selected: boolean) => React.ReactNode
  bulkActions: BulkAction[]
  emptyMessage?: string
}

export function SelectableTable<T>({
  items,
  getKey,
  columns,
  renderRow,
  bulkActions,
  emptyMessage,
}: SelectableTableProps<T>) {
  const { t } = useTranslation()
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const allSelected = items.length > 0 && selectedIds.size === items.length
  const someSelected = selectedIds.size > 0 && selectedIds.size < items.length

  const toggleAll = useCallback(() => {
    if (allSelected) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(items.map(getKey)))
    }
  }, [allSelected, items, getKey])

  const toggleOne = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const handleAction = useCallback(
    (action: BulkAction) => {
      action.onClick(Array.from(selectedIds))
      setSelectedIds(new Set())
    },
    [selectedIds]
  )

  return (
    <div className="relative">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10">
              <Checkbox
                checked={allSelected ? true : someSelected ? "indeterminate" : false}
                onCheckedChange={toggleAll}
                aria-label={t("common.selectAll")}
              />
            </TableHead>
            {columns}
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => {
            const key = getKey(item)
            const selected = selectedIds.has(key)
            return (
              <TableRow key={key} data-selected={selected || undefined}>
                <TableCell className="w-10" onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={selected}
                    onCheckedChange={() => toggleOne(key)}
                    aria-label={`Select ${key}`}
                  />
                </TableCell>
                {renderRow(item, selected)}
              </TableRow>
            )
          })}
          {items.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={100}
                className="text-center text-muted-foreground py-8"
              >
                {emptyMessage || t("common.noData")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {selectedIds.size > 0 && (
        <div className="sticky bottom-4 mx-4 mt-4 flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-3 shadow-lg">
          <span className="text-sm font-medium">
            {t("common.nSelected", { count: selectedIds.size })}
          </span>
          <div className="flex-1" />
          {bulkActions.map((action) => (
            <Button
              key={action.label}
              size="sm"
              variant={action.variant ?? "outline"}
              onClick={() => handleAction(action)}
            >
              {action.label}
            </Button>
          ))}
        </div>
      )}
    </div>
  )
}
