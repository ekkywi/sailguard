import { Route, Routes } from 'react-router-dom'
import { RequireAuth } from './features/auth/RequireAuth'
import LoginPage from './features/auth/LoginPage'
import DevicesPage from './features/devices/DevicesPage'
import DeviceDetailPage from './features/devices/DeviceDetailPage'
import TokensPage from './features/tokens/TokensPage'
import GroupsPage from './features/groups/GroupsPage'
import GroupDetailPage from './features/groups/GroupDetailPage'
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
        <Route path="/groups" element={<GroupsPage />} />
        <Route path="/groups/:id" element={<GroupDetailPage />} />
      </Route>
    </Routes>
  )
}
