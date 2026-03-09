import { useState, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useAdminUsers, useAdminMutations } from "@/hooks/use-admin"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
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
} from "@/components/ui/card"

const PAGE_SIZE = 20

export function UsersPage() {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [roleFilter, setRoleFilter] = useState("")
  const [page, setPage] = useState(0)
  const [editingUser, setEditingUser] = useState<{ id: string; role: string } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const { data, loading, error, refetch } = useAdminUsers(
    search || undefined,
    roleFilter || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )
  const { updateUserRole, deleteUser } = useAdminMutations()

  const handleRoleChange = useCallback(
    async (id: string, role: string) => {
      try {
        await updateUserRole(id, role)
        setEditingUser(null)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to update role")
      }
    },
    [updateUserRole, refetch]
  )

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteUser(id)
        setConfirmDelete(null)
        refetch()
      } catch (e) {
        alert(e instanceof Error ? e.message : "Failed to delete user")
      }
    },
    [deleteUser, refetch]
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t('adminUsers.title')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('adminUsers.usersRegistered', { count: total })}
        </p>
      </div>

      <div className="flex gap-3">
        <Input
          placeholder={t('adminUsers.searchPlaceholder')}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(0)
          }}
          className="max-w-sm"
        />
        <select
          value={roleFilter}
          onChange={(e) => {
            setRoleFilter(e.target.value)
            setPage(0)
          }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t('adminUsers.allRoles')}</option>
          <option value="user">user</option>
          <option value="provider">provider</option>
          <option value="admin">admin</option>
        </select>
      </div>

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">{t('adminUsers.loadingUsers')}</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('adminUsers.email')}</TableHead>
                    <TableHead>{t('adminUsers.displayName')}</TableHead>
                    <TableHead>{t('adminUsers.role')}</TableHead>
                    <TableHead>{t('adminUsers.createdAt')}</TableHead>
                    <TableHead className="text-right">{t('adminAgents.actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(data?.users ?? []).map((user) => (
                    <TableRow key={user.id}>
                      <TableCell className="font-mono text-xs">{user.email}</TableCell>
                      <TableCell>{user.display_name}</TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            user.role === "admin"
                              ? "destructive"
                              : user.role === "provider"
                              ? "secondary"
                              : "default"
                          }
                        >
                          {user.role}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {new Date(user.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-right space-x-2">
                        {editingUser?.id === user.id ? (
                          <span className="inline-flex gap-1">
                            {["user", "provider", "admin"].map((r) => (
                              <Button
                                key={r}
                                size="sm"
                                variant={r === editingUser.role ? "default" : "outline"}
                                onClick={() => handleRoleChange(user.id, r)}
                              >
                                {r}
                              </Button>
                            ))}
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => setEditingUser(null)}
                            >
                              {t('common.cancel')}
                            </Button>
                          </span>
                        ) : confirmDelete === user.id ? (
                          <span className="inline-flex gap-1">
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleDelete(user.id)}
                            >
                              {t('adminUsers.confirmDelete')}
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
                              onClick={() =>
                                setEditingUser({ id: user.id, role: user.role })
                              }
                            >
                              {t('adminUsers.editRole')}
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              className="text-destructive"
                              onClick={() => setConfirmDelete(user.id)}
                            >
                              {t('common.delete')}
                            </Button>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(data?.users ?? []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                        {t('adminUsers.noUsers')}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {t('common.page')} {page + 1} / {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t('common.previous')}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('common.next')}
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
