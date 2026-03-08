**English** | [中文](README_zh.md)

# PeerClaw Web App

The PeerClaw web application — an Agent Marketplace frontend built with React + TypeScript + Vite.

## Architecture

```
src/
├── api/           # API client, request utilities, type definitions
├── assets/        # Static assets
├── components/
│   ├── ui/        # Base UI components (shadcn/ui + Radix UI + TailwindCSS)
│   ├── agents/    # Agent management components
│   ├── auth/      # Login, register, auth guards
│   ├── layout/    # AppLayout, Sidebar, ConsoleLayout, PublicLayout
│   ├── provider/  # Provider console components
│   ├── public/    # Public marketplace components
│   ├── playground/# Agent playground & invocation UI
│   └── overview/  # Dashboard overview components
├── hooks/         # Custom React hooks (auth, provider, playground, etc.)
├── lib/           # Utility functions
└── pages/         # Page components (see Routes below)
```

## Routes

### Public (No Auth)

| Route | Page | Description |
|-------|------|-------------|
| `/` | LandingPage | Platform stats, value propositions, search |
| `/directory` | DirectoryPage | Agent marketplace with search, filter, sort |
| `/agents/:id` | PublicProfilePage | Agent profile, reputation, reviews |
| `/playground` | PlaygroundPage | Try agents via chat interface |
| `/playground/:agentId` | PlaygroundPage | Pre-selected agent playground |
| `/login` | LoginPage | User authentication |
| `/register` | RegisterPage | New user registration |

### Provider Console (Auth Required)

| Route | Page | Description |
|-------|------|-------------|
| `/console` | ProviderDashboardPage | Agent stats, claim tokens |
| `/console/publish` | AgentPublishPage | 5-step agent publish wizard |
| `/console/agents/:id` | ProviderAgentDetailPage | Agent details, analytics, contacts |
| `/console/agents/:id/edit` | AgentEditPage | Edit published agent |
| `/console/invocations` | InvocationHistoryPage | Invocation history |
| `/console/api-keys` | APIKeysPage | API key management |

### Admin Dashboard (Admin Role)

| Route | Page | Description |
|-------|------|-------------|
| `/admin` | OverviewPage | System stats & health |
| `/admin/users` | UsersPage | User management |
| `/admin/agents` | AgentsPage | Agent management & verification |
| `/admin/reports` | ReportsPage | Abuse report moderation |
| `/admin/categories` | CategoriesPage | Category CRUD |
| `/admin/analytics` | AnalyticsPage | Global invocation analytics |
| `/admin/invocations` | InvocationsPage | System-wide invocation logs |

## Tech Stack

- **React** 19 + **TypeScript** ~5.9
- **Vite** 7.3 (build & dev server)
- **TailwindCSS** 4.2 (styling)
- **Radix UI** (accessible components)
- **Lucide React** (icons)
- **Recharts** (analytics charts)
- **React Router** 7 (routing)

## Development

```bash
# Install dependencies
npm install

# Start dev server
npm run dev

# Type check + production build
npm run build

# Lint
npm run lint
```

## API Integration

The app communicates with the PeerClaw server via REST API at `/api/v1/*`. Key endpoint groups:

- `/auth/*` — User registration, login, JWT tokens, API keys
- `/directory/*` — Public agent directory, profiles, reviews
- `/invoke/:agentId` — Agent invocation (supports SSE streaming)
- `/provider/*` — Provider console (publish, manage, analytics)
- `/admin/*` — Admin dashboard (users, agents, reports, categories)
- `/claim-tokens` — Claim token generation for agent pairing
- `/blobs` — File upload/download
