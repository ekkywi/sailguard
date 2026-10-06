import { Link, Route, Routes } from 'react-router-dom';
import { RequireAuth } from './features/auth/RequireAuth';
import LoginPage from './features/auth/LoginPage';
import HomePage from './pages/HomePage';

export default function App() {
  return (
    <div className="shell">
      <header className="top">
        <Link to="/" className="brand">
          SailGuard
        </Link>
        <nav>
          <span className="muted">MVP</span>
        </nav>
      </header>
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
    </div>
  )
}