[English](README.md) | **中文**

# PeerClaw Web App

PeerClaw Web 应用 — 基于 React + TypeScript + Vite 构建的 Agent Marketplace 前端。

## 架构

```
src/
├── api/           # API 客户端、请求工具、类型定义
├── assets/        # 静态资源
├── components/
│   ├── ui/        # 基础 UI 组件（shadcn/ui + Radix UI + TailwindCSS）
│   ├── agents/    # Agent 管理组件
│   ├── auth/      # 登录、注册、认证守卫
│   ├── layout/    # AppLayout, Sidebar, ConsoleLayout, PublicLayout
│   ├── provider/  # Provider 控制台组件
│   ├── public/    # 公开 Marketplace 组件
│   ├── playground/# Agent Playground 与调用 UI
│   └── overview/  # Dashboard 概览组件
├── hooks/         # 自定义 React Hooks（auth, provider, playground 等）
├── lib/           # 工具函数
└── pages/         # 页面组件（路由见下表）
```

## 路由

### 公开页面（无需登录）

| 路由 | 页面 | 描述 |
|------|------|------|
| `/` | LandingPage | 平台统计、价值主张、搜索入口 |
| `/directory` | DirectoryPage | Agent 市场（搜索、筛选、排序） |
| `/agents/:id` | PublicProfilePage | Agent 详情、声誉、评价 |
| `/playground` | PlaygroundPage | 通过聊天界面试用 Agent |
| `/playground/:agentId` | PlaygroundPage | 预选 Agent 的 Playground |
| `/login` | LoginPage | 用户登录 |
| `/register` | RegisterPage | 用户注册 |

### Provider 控制台（需登录）

| 路由 | 页面 | 描述 |
|------|------|------|
| `/console` | ProviderDashboardPage | Agent 统计、Claim Token |
| `/console/publish` | AgentPublishPage | 5 步 Agent 发布向导 |
| `/console/agents/:id` | ProviderAgentDetailPage | Agent 详情、分析、联系人 |
| `/console/agents/:id/edit` | AgentEditPage | 编辑已发布的 Agent |
| `/console/invocations` | InvocationHistoryPage | 调用历史 |
| `/console/api-keys` | APIKeysPage | API Key 管理 |

### 管理后台（Admin 角色）

| 路由 | 页面 | 描述 |
|------|------|------|
| `/admin` | OverviewPage | 系统统计与健康状态 |
| `/admin/users` | UsersPage | 用户管理 |
| `/admin/agents` | AgentsPage | Agent 管理与验证 |
| `/admin/reports` | ReportsPage | 举报审核 |
| `/admin/categories` | CategoriesPage | 分类管理 |
| `/admin/analytics` | AnalyticsPage | 全局调用分析 |
| `/admin/invocations` | InvocationsPage | 全局调用日志 |

## 技术栈

- **React** 19 + **TypeScript** ~5.9
- **Vite** 7.3（构建与开发服务器）
- **TailwindCSS** 4.2（样式）
- **Radix UI**（无障碍组件）
- **Lucide React**（图标）
- **Recharts**（分析图表）
- **React Router** 7（路由）

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 类型检查 + 生产构建
npm run build

# 代码检查
npm run lint
```

## API 集成

应用通过 REST API 与 PeerClaw Server 通信，端点前缀为 `/api/v1/*`：

- `/auth/*` — 用户注册、登录、JWT 令牌、API Key
- `/directory/*` — 公开 Agent 目录、详情、评价
- `/invoke/:agentId` — Agent 调用（支持 SSE 流式）
- `/provider/*` — Provider 控制台（发布、管理、分析）
- `/admin/*` — 管理后台（用户、Agent、举报、分类）
- `/claim-tokens` — Claim Token 生成（Agent 配对）
- `/blobs` — 文件上传/下载
