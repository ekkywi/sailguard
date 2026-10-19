import { useEffect, useState } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { SailGuardLogo } from './SailGuardLogo'
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

function IconDownload() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M12 4v10m0 0 3.5-3.5M12 14 8.5 10.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M5 16.5V18a1.5 1.5 0 0 0 1.5 1.5h11A1.5 1.5 0 0 0 19 18v-1.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function IconPolicies() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M12 3.5 5.5 6.2v5.1c0 4.1 2.7 7.8 6.5 9.2 3.8-1.4 6.5-5.1 6.5-9.2V6.2L12 3.5Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <path
        d="M9.5 12.2 11.2 13.9 14.8 10"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconCategories() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M4.5 7.5h6.5v6.5H4.5V7.5Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <path
        d="M13 7.5h6.5v3H13v-3Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <path
        d="M13 13h6.5v3.5H13V13Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconEvents() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M5 6.5h14M5 12h10M5 17.5h12"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <circle cx="18.5" cy="12" r="1.75" fill="currentColor" />
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
              <SailGuardLogo />
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
            <NavLink to="/groups" className={navClass} title="Groups">
              <IconGroups />
              <span className="nav-label">Groups</span>
            </NavLink>
            <NavLink to="/tokens" className={navClass} title="Tokens">
              <IconTokens />
              <span className="nav-label">Tokens</span>
            </NavLink>
            <NavLink
              to="/agent-download"
              className={navClass}
              title="Download agent"
            >
              <IconDownload />
              <span className="nav-label">Download</span>
            </NavLink>
            <p className="sidebar-section">Control</p>
            <NavLink to="/policies" className={navClass} title="Policies">
              <IconPolicies />
              <span className="nav-label">Policies</span>
            </NavLink>
            <NavLink to="/categories" className={navClass} title="Categories">
              <IconCategories />
              <span className="nav-label">Categories</span>
            </NavLink>
            <p className="sidebar-section">Telemetry</p>
            <NavLink to="/events" className={navClass} title="Events">
              <IconEvents />
              <span className="nav-label">Events</span>
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
