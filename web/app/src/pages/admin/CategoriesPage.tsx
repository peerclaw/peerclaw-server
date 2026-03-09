import { useState, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useAdminCategories, useAdminMutations } from "@/hooks/use-admin"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { Category } from "@/api/types"

interface CategoryFormData {
  name: string
  slug: string
  description: string
  icon: string
  sort_order: number
}

const emptyForm: CategoryFormData = {
  name: "",
  slug: "",
  description: "",
  icon: "",
  sort_order: 0,
}

export function CategoriesPage() {
  const { t } = useTranslation()
  const { data, loading, error, refetch } = useAdminCategories()
  const { createCategory, updateCategory, deleteCategory } = useAdminMutations()
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<CategoryFormData>(emptyForm)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const handleSubmit = useCallback(async () => {
    try {
      if (editingId) {
        await updateCategory(editingId, form)
      } else {
        await createCategory(form)
      }
      setShowForm(false)
      setEditingId(null)
      setForm(emptyForm)
      refetch()
    } catch (e) {
      alert(e instanceof Error ? e.message : "Failed to save category")
    }
  }, [editingId, form, createCategory, updateCategory, refetch])

  const handleEdit = useCallback((cat: Category) => {
    setEditingId(cat.id)
    setForm({
      name: cat.name,
      slug: cat.slug,
      description: cat.description,
      icon: cat.icon,
      sort_order: cat.sort_order,
    })
    setShowForm(true)
  }, [])

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteCategory(id)
        setConfirmDelete(null)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to delete category")
      }
    },
    [deleteCategory, refetch]
  )

  const categories = data?.categories ?? []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('adminCategories.title')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('adminCategories.categoriesCount', { count: categories.length })}
          </p>
        </div>
        <Button
          onClick={() => {
            setEditingId(null)
            setForm(emptyForm)
            setShowForm(true)
          }}
        >
          {t('adminCategories.addCategory')}
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">
              {editingId ? t('adminCategories.editCategory') : t('adminCategories.newCategory')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm text-muted-foreground">{t('adminCategories.name')}</label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder={t('adminCategories.namePlaceholder')}
                />
              </div>
              <div>
                <label className="text-sm text-muted-foreground">{t('adminCategories.slug')}</label>
                <Input
                  value={form.slug}
                  onChange={(e) => setForm({ ...form, slug: e.target.value })}
                  placeholder={t('adminCategories.slugPlaceholder')}
                />
              </div>
              <div>
                <label className="text-sm text-muted-foreground">{t('adminCategories.icon')}</label>
                <Input
                  value={form.icon}
                  onChange={(e) => setForm({ ...form, icon: e.target.value })}
                  placeholder={t('adminCategories.iconPlaceholder')}
                />
              </div>
              <div>
                <label className="text-sm text-muted-foreground">{t('adminCategories.sortOrder')}</label>
                <Input
                  type="number"
                  value={form.sort_order}
                  onChange={(e) =>
                    setForm({ ...form, sort_order: parseInt(e.target.value) || 0 })
                  }
                />
              </div>
              <div className="col-span-2">
                <label className="text-sm text-muted-foreground">{t('adminCategories.description')}</label>
                <Input
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  placeholder={t('adminCategories.descPlaceholder')}
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Button onClick={handleSubmit}>
                {editingId ? t('common.update') : t('common.create')}
              </Button>
              <Button
                variant="ghost"
                onClick={() => {
                  setShowForm(false)
                  setEditingId(null)
                  setForm(emptyForm)
                }}
              >
                {t('common.cancel')}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('adminCategories.loadingCategories')}</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('adminCategories.name')}</TableHead>
                  <TableHead>{t('adminCategories.slug')}</TableHead>
                  <TableHead>{t('adminCategories.description')}</TableHead>
                  <TableHead>{t('adminCategories.icon')}</TableHead>
                  <TableHead>{t('adminCategories.sortOrder')}</TableHead>
                  <TableHead className="text-right">{t('adminAgents.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categories.map((cat) => (
                  <TableRow key={cat.id}>
                    <TableCell className="font-medium">{cat.name}</TableCell>
                    <TableCell className="font-mono text-xs">{cat.slug}</TableCell>
                    <TableCell className="max-w-[200px] truncate text-muted-foreground">
                      {cat.description}
                    </TableCell>
                    <TableCell>{cat.icon}</TableCell>
                    <TableCell>{cat.sort_order}</TableCell>
                    <TableCell className="text-right space-x-1">
                      {confirmDelete === cat.id ? (
                        <span className="inline-flex gap-1">
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={() => handleDelete(cat.id)}
                          >
                            {t('common.confirm')}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => setConfirmDelete(null)}
                          >
                            {t('common.cancel')}
                          </Button>
                        </span>
                      ) : (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleEdit(cat)}
                          >
                            {t('common.edit')}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            className="text-destructive"
                            onClick={() => setConfirmDelete(cat.id)}
                          >
                            {t('common.delete')}
                          </Button>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {categories.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      {t('adminCategories.noCategories')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
