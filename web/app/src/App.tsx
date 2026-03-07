import { Routes, Route } from "react-router-dom"
import { AppLayout } from "@/components/layout/AppLayout"
import { PublicLayout } from "@/components/public/PublicLayout"
import { OverviewPage } from "@/pages/OverviewPage"
import { AgentsPage } from "@/pages/AgentsPage"
import { AgentDetailPage } from "@/pages/AgentDetailPage"
import { LandingPage } from "@/pages/LandingPage"
import { DirectoryPage } from "@/pages/DirectoryPage"
import { PublicProfilePage } from "@/pages/PublicProfilePage"

export function App() {
  return (
    <div className="dark">
      <Routes>
        {/* Public routes */}
        <Route element={<PublicLayout />}>
          <Route index element={<LandingPage />} />
          <Route path="directory" element={<DirectoryPage />} />
          <Route path="agents/:id" element={<PublicProfilePage />} />
        </Route>

        {/* Admin routes */}
        <Route path="admin" element={<AppLayout />}>
          <Route index element={<OverviewPage />} />
          <Route path="agents" element={<AgentsPage />} />
          <Route path="agents/:id" element={<AgentDetailPage />} />
        </Route>
      </Routes>
    </div>
  )
}
