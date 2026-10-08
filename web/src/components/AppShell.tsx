import { useEffect, useState } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { ThemeToggle } from './ThemeToggle'
import { UserMenu } from './UserMenu'

const SIDEBAR_KEY = 'sg_sidebar_collapsed'

type AppShellProps = {
  children: React.ReactNode
  userLabel?: string | null
}

function navClass({ isActive }: { isActive: boolean }) {
  return isActive ? 'nav-link nav-link-active' : 'nav-link'
}

function IconOverview() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1v-9.5Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconDevices() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect
        x="3.75"
        y="5.75"
        width="16.5"
        height="11.5"
        rx="1.5"
        stroke="currentColor"
        strokeWidth="1.75"
      />
      <path
        d="M8 19.5h8"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function IconGroups() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="9" cy="8" r="2.5" stroke="currentColor" strokeWidth="1.75" />
      <circle cx="16" cy="9.5" r="2" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M3.5 18.5c.6-2.6 2.7-4 5.5-4s4.9 1.4 5.5 4"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <path
        d="M14 14.5c1.8.2 3.3 1.2 3.9 3"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function IconTokens() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="12" cy="12" r="7.25" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M12 8.5v7M9.5 10.5c.4-.7 1.2-1.1 2.5-1.1 1.5 0 2.5.7 2.5 1.8S13.5 13 12 13s-2.5.5-2.5 1.6c0 1.1 1.1 1.9 2.7 1.9 1.2 0 2-.4 2.4-1"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconPanelLeft() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect
        x="3.75"
        y="4.75"
        width="16.5"
        height="14.5"
        rx="2"
        stroke="currentColor"
        strokeWidth="1.75"
      />
      <path d="M9 5v14" stroke="currentColor" strokeWidth="1.75" />
    </svg>
  )
}

export function AppShell({ children, userLabel }: AppShellProps) {
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem(SIDEBAR_KEY) === '1'
    } catch {
      return false
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem(SIDEBAR_KEY, collapsed ? '1' : '0')
    } catch {
      /* ignore */
    }
  }, [collapsed])

  return (
    <div className={collapsed ? 'shell shell-sidebar-collapsed' : 'shell'}>
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <aside className="sidebar" aria-label="Sidebar">
        <div className="sidebar-top">
          <div className="sidebar-brand-row">
            <Link to="/" className="brand" title="SailGuard">
              <span className="brand-full">SailGuard</span>
            </Link>
            <button
              type="button"
              className="btn btn-icon sidebar-toggle"
              onClick={() => setCollapsed((v) => !v)}
              aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              aria-expanded={!collapsed}
            >
              <IconPanelLeft />
            </button>
          </div>
          <nav className="sidebar-nav" aria-label="Main">
            <p className="sidebar-section">Workspace</p>
            <NavLink to="/" end className={navClass} title="Overview">
              <IconOverview />
              <span className="nav-label">Overview</span>
            </NavLink>
            <p className="sidebar-section">Inventory</p>
            <NavLink to="/devices" className={navClass} title="Devices">
              <IconDevices />
              <span className="nav-label">Devices</span>
            </NavLink>
            <span
              className="nav-link nav-link-disabled"
              title="Groups (coming soon)"
            >
              <IconGroups />
              <span className="nav-label">Groups</span>
            </span>
            <NavLink to="/tokens" className={navClass} title="Tokens">
              <IconTokens />
              <span className="nav-label">Tokens</span>
            </NavLink>
          </nav>
        </div>
      </aside>
      <div className="shell-body">
        <header className="topbar">
          <div className="topbar-spacer" />
          <ThemeToggle />
          <UserMenu userLabel={userLabel} />
        </header>
        <main id="main-content" className="shell-main" tabIndex={-1}>
          {children}
        </main>
      </div>
    </div>
  )
}
