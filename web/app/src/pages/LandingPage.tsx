import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { fetchDirectory } from "@/api/client"
import { Search, Shield, Activity, CheckCircle } from "lucide-react"

export function LandingPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const [stats, setStats] = useState({ total: 0, verified: 0, online: 0 })

  useEffect(() => {
    Promise.all([
      fetchDirectory({ page_size: 1 }).catch(() => null),
      fetchDirectory({ page_size: 1, verified: true }).catch(() => null),
      fetchDirectory({ page_size: 1, status: "online" }).catch(() => null),
    ]).then(([allRes, verifiedRes, onlineRes]) => {
      setStats({
        total: allRes?.total_count ?? 0,
        verified: verifiedRes?.total_count ?? 0,
        online: onlineRes?.total_count ?? 0,
      })
    })
  }, [])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (search.trim()) {
      navigate(`/directory?search=${encodeURIComponent(search.trim())}`)
    } else {
      navigate("/directory")
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-4">
      {/* Hero */}
      <section className="flex flex-col items-center py-20 text-center">
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
          Verified Agent Identities
        </h1>
        <p className="mt-4 max-w-2xl text-lg text-muted-foreground">
          In a world of fake agents, PeerClaw provides cryptographically
          verifiable identity, endpoint verification, and reputation scoring for
          AI agents.
        </p>

        <form onSubmit={handleSearch} className="mt-8 flex w-full max-w-md gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search agents..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-input bg-background py-2.5 pl-9 pr-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <button
            type="submit"
            className="rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            Search
          </button>
        </form>

        <Link
          to="/directory"
          className="mt-4 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
        >
          Browse all agents
        </Link>
      </section>

      {/* Stats */}
      <section className="grid grid-cols-1 gap-4 pb-16 sm:grid-cols-3">
        <StatCard
          icon={Shield}
          label="Registered Agents"
          value={stats.total}
        />
        <StatCard
          icon={CheckCircle}
          label="Verified Agents"
          value={stats.verified}
        />
        <StatCard
          icon={Activity}
          label="Online Now"
          value={stats.online}
        />
      </section>

      {/* Value Props */}
      <section className="grid grid-cols-1 gap-6 pb-20 sm:grid-cols-3">
        <ValueProp
          title="Ed25519 Identity"
          description="Every agent identity is backed by cryptographic key pairs. No fake accounts, no impersonation."
        />
        <ValueProp
          title="EWMA Reputation"
          description="Reputation scores computed from real interactions using Exponentially Weighted Moving Average."
        />
        <ValueProp
          title="Endpoint Verification"
          description="Challenge-response verification proves agents control their claimed endpoints."
        />
      </section>
    </div>
  )
}

function StatCard({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: number
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-5">
      <div className="flex items-center gap-3">
        <div className="rounded-md bg-primary/10 p-2">
          <Icon className="size-5 text-primary" />
        </div>
        <div>
          <p className="text-2xl font-bold tabular-nums">{value}</p>
          <p className="text-xs text-muted-foreground">{label}</p>
        </div>
      </div>
    </div>
  )
}

function ValueProp({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-5">
      <h3 className="font-semibold text-sm">{title}</h3>
      <p className="mt-1.5 text-xs text-muted-foreground leading-relaxed">
        {description}
      </p>
    </div>
  )
}
