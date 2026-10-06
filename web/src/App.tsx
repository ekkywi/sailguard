import { Route, Routes } from 'react-router-dom'
import { RequireAuth } from './features/auth/RequireAuth'
import LoginPage from './features/auth/LoginPage'
import HomePage from './pages/HomePage'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <HomePage />
          </RequireAuth>
        }
      />
    </Routes>
  )
}
