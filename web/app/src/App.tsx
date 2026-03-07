import { Routes, Route } from "react-router-dom"
import { AppLayout } from "@/components/layout/AppLayout"
import { ConsoleLayout } from "@/components/layout/ConsoleLayout"
import { PublicLayout } from "@/components/public/PublicLayout"
import { AuthProvider } from "@/hooks/use-auth"
import { AuthGuard } from "@/components/auth/AuthGuard"
import { AdminGuard } from "@/components/auth/AdminGuard"
import { OverviewPage } from "@/pages/OverviewPage"
import { AgentsPage } from "@/pages/AgentsPage"
import { AgentDetailPage } from "@/pages/AgentDetailPage"
import { LandingPage } from "@/pages/LandingPage"
import { DirectoryPage } from "@/pages/DirectoryPage"
import { PublicProfilePage } from "@/pages/PublicProfilePage"
import { LoginPage } from "@/pages/LoginPage"
import { RegisterPage } from "@/pages/RegisterPage"
import { PlaygroundPage } from "@/pages/PlaygroundPage"
import { ProviderDashboardPage } from "@/pages/ProviderDashboardPage"
import { AgentPublishPage } from "@/pages/AgentPublishPage"
import { ProviderAgentDetailPage } from "@/pages/ProviderAgentDetailPage"
import { InvocationHistoryPage } from "@/pages/InvocationHistoryPage"
import { APIKeysPage } from "@/pages/APIKeysPage"
import { UsersPage } from "@/pages/admin/UsersPage"
import { ReportsPage } from "@/pages/admin/ReportsPage"
import { CategoriesPage } from "@/pages/admin/CategoriesPage"
import { AnalyticsPage } from "@/pages/admin/AnalyticsPage"
import { InvocationsPage } from "@/pages/admin/InvocationsPage"

export function App() {
  return (
    <div className="dark">
      <AuthProvider>
        <Routes>
          {/* Public routes */}
          <Route element={<PublicLayout />}>
            <Route index element={<LandingPage />} />
            <Route path="directory" element={<DirectoryPage />} />
            <Route path="agents/:id" element={<PublicProfilePage />} />
            <Route path="login" element={<LoginPage />} />
            <Route path="register" element={<RegisterPage />} />
            <Route path="playground" element={<PlaygroundPage />} />
            <Route path="playground/:agentId" element={<PlaygroundPage />} />
          </Route>

          {/* Provider console routes (auth required) */}
          <Route
            path="console"
            element={
              <AuthGuard>
                <ConsoleLayout />
              </AuthGuard>
            }
          >
            <Route index element={<ProviderDashboardPage />} />
            <Route path="publish" element={<AgentPublishPage />} />
            <Route path="agents/:id" element={<ProviderAgentDetailPage />} />
            <Route path="invocations" element={<InvocationHistoryPage />} />
            <Route path="api-keys" element={<APIKeysPage />} />
          </Route>

          {/* Admin routes (admin role required) */}
          <Route
            path="admin"
            element={
              <AdminGuard>
                <AppLayout />
              </AdminGuard>
            }
          >
            <Route index element={<OverviewPage />} />
            <Route path="users" element={<UsersPage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="agents/:id" element={<AgentDetailPage />} />
            <Route path="reports" element={<ReportsPage />} />
            <Route path="categories" element={<CategoriesPage />} />
            <Route path="analytics" element={<AnalyticsPage />} />
            <Route path="invocations" element={<InvocationsPage />} />
          </Route>
        </Routes>
      </AuthProvider>
    </div>
  )
}
