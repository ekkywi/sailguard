import { Route, Routes } from 'react-router-dom'
import { RequireAuth } from './features/auth/RequireAuth'
import LoginPage from './features/auth/LoginPage'
import DevicesPage from './features/devices/DevicesPage'
import DeviceDetailPage from './features/devices/DeviceDetailPage'
import TokensPage from './features/tokens/TokensPage'
import GroupsPage from './features/groups/GroupsPage'
import GroupDetailPage from './features/groups/GroupDetailPage'
import PoliciesPage from './features/policies/PoliciesPage'
import PolicyDetailPage from './features/policies/PolicyDetailPage'
import CategoriesPage from './features/categories/CategoriesPage'
import CategoryDetailPage from './features/categories/CategoryDetailPage'
import EventsPage from './features/events/EventsPage'
import AgentDownloadPage from './features/agent-packages/AgentDownloadPage'
import UsersPage from './features/users/UsersPage'
import AuditLogsPage from './features/audit/AuditLogsPage'
import AppLayout from './layouts/AppLayout'
import HomePage from './pages/HomePage'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAuth>
            <AppLayout />
          </RequireAuth>
        }
      >
        <Route path="/" element={<HomePage />} />
        <Route path="/devices" element={<DevicesPage />} />
        <Route path="/devices/:id" element={<DeviceDetailPage />} />
        <Route path="/tokens" element={<TokensPage />} />
        <Route path="/agent-download" element={<AgentDownloadPage />} />
        <Route path="/groups" element={<GroupsPage />} />
        <Route path="/groups/:id" element={<GroupDetailPage />} />
        <Route path="/policies" element={<PoliciesPage />} />
        <Route path="/policies/:id" element={<PolicyDetailPage />} />
        <Route path="/categories" element={<CategoriesPage />} />
        <Route path="/categories/:code" element={<CategoryDetailPage />} />
        <Route path="/events" element={<EventsPage />} />
        <Route path="/users" element={<UsersPage />} />
        <Route path="/audit" element={<AuditLogsPage />} />
      </Route>
    </Routes>
  )
}
