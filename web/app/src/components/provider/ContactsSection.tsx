import { useState } from "react"
import {
  useAgentContacts,
  useAgentContactMutations,
} from "@/hooks/use-provider"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Trash2 } from "lucide-react"

interface ContactsSectionProps {
  agentId: string
}

export function ContactsSection({ agentId }: ContactsSectionProps) {
  const { data, loading, error, refetch } = useAgentContacts(agentId)
  const { addContact, removeContact } = useAgentContactMutations(agentId)
  const [contactId, setContactId] = useState("")
  const [alias, setAlias] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!contactId.trim()) return

    setSubmitting(true)
    setFormError(null)
    try {
      await addContact(contactId.trim(), alias.trim())
      setContactId("")
      setAlias("")
      refetch()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to add contact")
    } finally {
      setSubmitting(false)
    }
  }

  const handleRemove = async (contactAgentId: string) => {
    try {
      await removeContact(contactAgentId)
      refetch()
    } catch (err) {
      setFormError(
        err instanceof Error ? err.message : "Failed to remove contact"
      )
    }
  }

  const contacts = data?.contacts ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          Contacts Whitelist
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-xs text-muted-foreground">
          Only agents listed here can send messages to this agent.
        </p>

        {/* Add contact form */}
        <form onSubmit={handleAdd} className="flex gap-2">
          <Input
            placeholder="Contact Agent ID"
            value={contactId}
            onChange={(e) => setContactId(e.target.value)}
            className="flex-1"
          />
          <Input
            placeholder="Alias (optional)"
            value={alias}
            onChange={(e) => setAlias(e.target.value)}
            className="w-40"
          />
          <Button type="submit" size="sm" disabled={submitting || !contactId.trim()}>
            {submitting ? "Adding..." : "Add"}
          </Button>
        </form>

        {formError && <p className="text-xs text-destructive">{formError}</p>}
        {error && <p className="text-xs text-destructive">{error}</p>}

        {/* Contacts table */}
        {loading ? (
          <p className="text-xs text-muted-foreground">Loading contacts...</p>
        ) : contacts.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            No contacts yet. Add an agent ID to allow it to communicate with
            this agent.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Agent ID</TableHead>
                <TableHead>Alias</TableHead>
                <TableHead>Added</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {contacts.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-mono text-xs">
                    {c.contact_agent_id}
                  </TableCell>
                  <TableCell className="text-sm">
                    {c.alias || "-"}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(c.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemove(c.contact_agent_id)}
                    >
                      <Trash2 className="size-3" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
