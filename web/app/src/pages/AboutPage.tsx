import { useTranslation } from "react-i18next"
import {
  Fingerprint,
  Network,
  ShieldCheck,
  UserPlus,
  BadgeCheck,
  TrendingUp,
  Search,
  ArrowRightLeft,
  Handshake,
  Github,
} from "lucide-react"

export function AboutPage() {
  const { t } = useTranslation()

  const values = [
    { icon: Fingerprint, titleKey: "about.valueIdentityTitle", descKey: "about.valueIdentityDesc" },
    { icon: Network, titleKey: "about.valueInteropTitle", descKey: "about.valueInteropDesc" },
    { icon: ShieldCheck, titleKey: "about.valueTrustTitle", descKey: "about.valueTrustDesc" },
  ]

  const steps = [
    { icon: UserPlus, titleKey: "about.stepRegisterTitle", descKey: "about.stepRegisterDesc" },
    { icon: BadgeCheck, titleKey: "about.stepVerifyTitle", descKey: "about.stepVerifyDesc" },
    { icon: TrendingUp, titleKey: "about.stepEarnTitle", descKey: "about.stepEarnDesc" },
    { icon: Search, titleKey: "about.stepDiscoverTitle", descKey: "about.stepDiscoverDesc" },
    { icon: ArrowRightLeft, titleKey: "about.stepBridgeTitle", descKey: "about.stepBridgeDesc" },
    { icon: Handshake, titleKey: "about.stepTrustTitle", descKey: "about.stepTrustDesc" },
  ]

  const phases: { titleKey: string; descKey: string; status: "done" | "current" | "upcoming" }[] = [
    { titleKey: "about.phase1Title", descKey: "about.phase1Desc", status: "done" },
    { titleKey: "about.phase2Title", descKey: "about.phase2Desc", status: "done" },
    { titleKey: "about.phase3Title", descKey: "about.phase3Desc", status: "done" },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4">
      {/* Hero */}
      <section className="flex flex-col items-center py-20 text-center">
        <img
          src="/logo.jpg"
          alt="PeerClaw"
          className="size-28 rounded-2xl object-cover mb-6"
        />
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
          {t("about.title")}
        </h1>
        <p className="mt-4 max-w-2xl text-lg text-muted-foreground">
          {t("about.subtitle")}
        </p>
      </section>

      {/* Why PeerClaw */}
      <section className="pb-16">
        <h2 className="text-2xl font-bold tracking-tight text-center mb-8">
          {t("about.whyTitle")}
        </h2>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="font-semibold text-sm text-destructive">
              {t("about.problemLabel")}
            </h3>
            <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
              {t("about.whyProblem")}
            </p>
          </div>
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="font-semibold text-sm text-primary">
              {t("about.solutionLabel")}
            </h3>
            <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
              {t("about.whySolution")}
            </p>
          </div>
        </div>
      </section>

      {/* Core Values */}
      <section className="pb-16">
        <h2 className="text-2xl font-bold tracking-tight text-center mb-8">
          {t("about.valuesTitle")}
        </h2>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          {values.map((v) => (
            <div
              key={v.titleKey}
              className="rounded-lg border border-border bg-card p-5"
            >
              <div className="rounded-md bg-primary/10 p-2 w-fit mb-3">
                <v.icon className="size-5 text-primary" />
              </div>
              <h3 className="font-semibold text-sm">{t(v.titleKey)}</h3>
              <p className="mt-1.5 text-xs text-muted-foreground leading-relaxed">
                {t(v.descKey)}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* How It Works */}
      <section className="pb-16">
        <h2 className="text-2xl font-bold tracking-tight text-center mb-8">
          {t("about.howTitle")}
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {steps.map((s, i) => (
            <div
              key={s.titleKey}
              className="rounded-lg border border-border bg-card p-5 flex gap-4"
            >
              <div className="flex-shrink-0">
                <div className="rounded-full bg-primary/10 size-10 flex items-center justify-center">
                  <span className="text-xs font-bold text-primary">
                    {i + 1}
                  </span>
                </div>
              </div>
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <s.icon className="size-4 text-muted-foreground" />
                  <h3 className="font-semibold text-sm">{t(s.titleKey)}</h3>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  {t(s.descKey)}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Roadmap */}
      <section className="pb-16">
        <h2 className="text-2xl font-bold tracking-tight text-center mb-8">
          {t("about.roadmapTitle")}
        </h2>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          {phases.map((p) => (
            <div
              key={p.titleKey}
              className={`rounded-lg border p-5 ${
                p.status === "current"
                  ? "border-primary bg-primary/5"
                  : "border-border bg-card"
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span
                  className={`size-2.5 rounded-full ${
                    p.status === "done"
                      ? "bg-green-500"
                      : p.status === "current"
                        ? "bg-primary animate-pulse"
                        : "bg-muted-foreground/30"
                  }`}
                />
                <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                  {t(`about.status_${p.status}`)}
                </span>
              </div>
              <h3 className="font-semibold text-sm">{t(p.titleKey)}</h3>
              <p className="mt-1.5 text-xs text-muted-foreground leading-relaxed">
                {t(p.descKey)}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* Open Source CTA */}
      <section className="pb-20">
        <div className="rounded-lg border border-border bg-card p-8 text-center">
          <div className="mx-auto rounded-full bg-primary/10 size-12 flex items-center justify-center mb-4">
            <Github className="size-6 text-primary" />
          </div>
          <h2 className="text-xl font-bold">{t("about.openSourceTitle")}</h2>
          <p className="mt-2 text-sm text-muted-foreground max-w-lg mx-auto">
            {t("about.openSourceDesc")}
          </p>
          <a
            href="https://github.com/peerclaw/peerclaw"
            target="_blank"
            rel="noopener noreferrer"
            className="mt-4 inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <Github className="size-4" />
            {t("about.viewOnGithub")}
          </a>
        </div>
      </section>
    </div>
  )
}
