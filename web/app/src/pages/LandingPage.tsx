import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { fetchDirectory } from "@/api/client"
import { Search, Shield, Activity, CheckCircle, ArrowRight, Fingerprint, BarChart3, Globe, Blocks } from "lucide-react"

export function LandingPage() {
  const navigate = useNavigate()
  const { t } = useTranslation()
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
    <div className="relative">
      {/* ─── Hero Section ─── */}
      <section className="relative overflow-hidden">
        {/* Animated gradient orbs */}
        <div className="pointer-events-none absolute inset-0">
          <div
            className="animate-float-1 absolute -top-32 -left-32 size-[500px] rounded-full opacity-30"
            style={{ background: "radial-gradient(circle, oklch(0.72 0.15 192 / 0.15), transparent 70%)" }}
          />
          <div
            className="animate-float-2 absolute -top-16 -right-24 size-[400px] rounded-full opacity-25"
            style={{ background: "radial-gradient(circle, oklch(0.65 0.18 285 / 0.12), transparent 70%)" }}
          />
          <div
            className="animate-float-3 absolute -bottom-24 left-1/3 size-[350px] rounded-full opacity-20"
            style={{ background: "radial-gradient(circle, oklch(0.72 0.15 192 / 0.1), transparent 70%)" }}
          />
        </div>

        {/* Dot grid overlay */}
        <div className="pointer-events-none absolute inset-0 dot-grid" />

        {/* Content */}
        <div className="relative mx-auto max-w-6xl px-4 pb-20 pt-24">
          <div className="flex flex-col items-center text-center">
            {/* Logo mark */}
            <div className="mb-8 flex items-center gap-3">
              <img
                src="/logo.jpg"
                alt="PeerClaw"
                className="size-14 rounded-2xl object-cover ring-2 ring-primary/20 shadow-[0_0_30px_oklch(0.72_0.15_192_/_0.15)]"
              />
            </div>

            {/* Headline */}
            <h1 className="max-w-3xl text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
              <span className="gradient-text">{t('landing.heroTitle')}</span>
            </h1>

            {/* Subtitle */}
            <p className="mt-5 max-w-xl text-base text-muted-foreground leading-relaxed sm:text-lg">
              {t('landing.heroDescription')}
            </p>

            {/* Search bar */}
            <form onSubmit={handleSearch} className="mt-10 flex w-full max-w-lg gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <input
                  type="text"
                  placeholder={t('landing.searchPlaceholder')}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full rounded-xl border border-border/60 bg-card/50 py-3 pl-10 pr-4 text-sm backdrop-blur-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary/40 transition-all"
                />
              </div>
              <button
                type="submit"
                className="rounded-xl bg-primary px-5 py-3 text-sm font-semibold text-primary-foreground transition-all hover:bg-primary/90 hover:shadow-[0_0_20px_oklch(0.72_0.15_192_/_0.25)]"
              >
                {t('common.search')}
              </button>
            </form>

            <Link
              to="/directory"
              className="mt-4 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-primary"
            >
              {t('landing.browseAll')}
              <ArrowRight className="size-3.5" />
            </Link>
          </div>
        </div>
      </section>

      {/* ─── Stats ─── */}
      <section className="relative mx-auto max-w-6xl px-4 pb-20">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <StatCard
            icon={Shield}
            label={t('landing.registeredAgents')}
            value={stats.total}
            accentColor="192"
          />
          <StatCard
            icon={CheckCircle}
            label={t('landing.verifiedAgents')}
            value={stats.verified}
            accentColor="160"
          />
          <StatCard
            icon={Activity}
            label={t('landing.onlineNow')}
            value={stats.online}
            accentColor="285"
          />
        </div>
      </section>

      {/* ─── Value Propositions ─── */}
      <section className="relative mx-auto max-w-6xl px-4 pb-24">
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
          <ValueProp
            icon={Fingerprint}
            title={t('landing.ed25519Title')}
            description={t('landing.ed25519Desc')}
          />
          <ValueProp
            icon={BarChart3}
            title={t('landing.ewmaTitle')}
            description={t('landing.ewmaDesc')}
          />
          <ValueProp
            icon={Globe}
            title={t('landing.endpointTitle')}
            description={t('landing.endpointDesc')}
          />
        </div>
      </section>

      {/* ─── Supported Platforms ─── */}
      <section className="relative mx-auto max-w-6xl px-4 pb-24">
        <div className="flex flex-col items-center text-center mb-10">
          <div className="flex size-10 items-center justify-center rounded-lg bg-primary/8 mb-4">
            <Blocks className="size-5 text-primary/80" />
          </div>
          <h2 className="text-2xl font-bold tracking-tight">{t('landing.platformsTitle')}</h2>
          <p className="mt-2 text-sm text-muted-foreground max-w-lg">{t('landing.platformsDesc')}</p>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {([
            { name: "OpenClaw", tech: "TypeScript", descKey: "landing.platformOpenClaw", color: "192" },
            { name: "IronClaw", tech: "Rust / WASM", descKey: "landing.platformIronClaw", color: "25" },
            { name: "PicoClaw", tech: "Go", descKey: "landing.platformPicoClaw", color: "160" },
            { name: "NanoBot", tech: "Python", descKey: "landing.platformNanoBot", color: "285" },
          ] as const).map((p) => (
            <div
              key={p.name}
              className="group rounded-xl border border-border/60 bg-card p-5 transition-all duration-300 hover:border-primary/20 hover:shadow-[0_0_30px_oklch(0.72_0.15_192_/_0.04)]"
            >
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-semibold text-sm">{p.name}</h3>
                <span
                  className="rounded-md px-2 py-0.5 text-[10px] font-medium"
                  style={{ background: `oklch(0.72 0.15 ${p.color} / 0.1)`, color: `oklch(0.72 0.15 ${p.color})` }}
                >
                  {p.tech}
                </span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">{t(p.descKey)}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

function StatCard({
  icon: Icon,
  label,
  value,
  accentColor,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: number
  accentColor: string
}) {
  return (
    <div
      className="group relative overflow-hidden rounded-xl border border-border/60 bg-card p-5 transition-all duration-300 hover:border-primary/20"
      style={{ boxShadow: `inset 0 1px 0 0 oklch(0.72 0.15 ${accentColor} / 0.08)` }}
    >
      <div className="flex items-center gap-4">
        <div
          className="flex size-11 items-center justify-center rounded-xl transition-colors"
          style={{ background: `oklch(0.72 0.15 ${accentColor} / 0.08)`, color: `oklch(0.72 0.15 ${accentColor})` }}
        >
          <Icon className="size-5" />
        </div>
        <div>
          <p className="text-2xl font-bold tabular-nums tracking-tight">{value}</p>
          <p className="text-xs text-muted-foreground font-medium">{label}</p>
        </div>
      </div>
    </div>
  )
}

function ValueProp({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: string
}) {
  return (
    <div className="group relative rounded-xl border border-border/60 bg-card p-6 transition-all duration-300 hover:border-primary/20 hover:shadow-[0_0_30px_oklch(0.72_0.15_192_/_0.04)]">
      <div className="flex size-10 items-center justify-center rounded-lg bg-primary/8 mb-4 transition-colors group-hover:bg-primary/12">
        <Icon className="size-5 text-primary/80" />
      </div>
      <h3 className="font-semibold text-sm">{title}</h3>
      <p className="mt-2 text-xs text-muted-foreground leading-relaxed">
        {description}
      </p>
    </div>
  )
}
