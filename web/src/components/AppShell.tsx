import { Link, useNavigate } from 'react-router-dom'
import { clearToken } from '../lib/auth-storage'
import { ThemeToggle } from './ThemeToggle'

type AppShellProps = {
  children: React.ReactNode
  userLabel?: string | null
}

export function AppShell({ children, userLabel }: AppShellProps) {
  const navigate = useNavigate()

  function logout() {
    clearToken()
    navigate('/login', { replace: true })
  }

  return (
    <div className="shell">
      <header className="topbar">
        <Link to="/" className="brand">
          SailGuard
        </Link>
        <div className="topbar-meta">
          {userLabel ? <span className="topbar-user">{userLabel}</span> : null}
          <ThemeToggle />
          <button type="button" className="btn btn-ghost" onClick={logout}>
            Sign out
          </button>
        </div>
      </header>
      <div className="shell-main">{children}</div>
    </div>
  )
}
