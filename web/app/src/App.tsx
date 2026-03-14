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
import { VerifyEmailPage } from "@/pages/VerifyEmailPage"
import { ForgotPasswordPage } from "@/pages/ForgotPasswordPage"
import { PlaygroundPage } from "@/pages/PlaygroundPage"
import { AboutPage } from "@/pages/AboutPage"
import { ProviderDashboardPage } from "@/pages/ProviderDashboardPage"
import { ProviderAgentsPage } from "@/pages/ProviderAgentsPage"
import { DiscoverAgentsPage } from "@/pages/DiscoverAgentsPage"
import { ProviderAgentDetailPage } from "@/pages/ProviderAgentDetailPage"
import { AgentEditPage } from "@/pages/AgentEditPage"
import { InvocationHistoryPage } from "@/pages/InvocationHistoryPage"
import { AccessRequestsPage } from "@/pages/AccessRequestsPage"
import { APIKeysPage } from "@/pages/APIKeysPage"
import { NotificationsPage } from "@/pages/NotificationsPage"
import { ProfilePage } from "@/pages/ProfilePage"
import { UsersPage } from "@/pages/admin/UsersPage"
import { ReportsPage } from "@/pages/admin/ReportsPage"
import { CategoriesPage } from "@/pages/admin/CategoriesPage"
import { AnalyticsPage } from "@/pages/admin/AnalyticsPage"
import { InvocationsPage } from "@/pages/admin/InvocationsPage"
import { NotFoundPage } from "@/pages/NotFoundPage"

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
            <Route path="verify-email" element={<VerifyEmailPage />} />
            <Route path="forgot-password" element={<ForgotPasswordPage />} />
            <Route path="playground" element={<PlaygroundPage />} />
            <Route path="playground/:agentId" element={<PlaygroundPage />} />
            <Route path="about" element={<AboutPage />} />
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
            <Route path="discover" element={<DiscoverAgentsPage />} />
            <Route path="agents" element={<ProviderAgentsPage />} />
            <Route path="agents/:id" element={<ProviderAgentDetailPage />} />
            <Route path="agents/:id/edit" element={<AgentEditPage />} />
            <Route path="invocations" element={<InvocationHistoryPage />} />
            <Route path="access-requests" element={<AccessRequestsPage />} />
            <Route path="api-keys" element={<APIKeysPage />} />
            <Route path="notifications" element={<NotificationsPage />} />
            <Route path="profile" element={<ProfilePage />} />
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

          {/* Catch-all 404 */}
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </AuthProvider>
    </div>
  )
}
