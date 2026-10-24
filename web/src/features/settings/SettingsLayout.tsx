import { NavLink, Navigate, Outlet, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { hasAnyPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'

function settingsNavClass({ isActive }: { isActive: boolean }) {
  return isActive ? 'settings-nav-link settings-nav-link-on' : 'settings-nav-link'
}

export default function SettingsLayout() {
  useDocumentTitle('Settings')
  const ctx = useOutletContext<AppOutletContext>()
  const canSee = hasAnyPermission(ctx.me, ['alert.read', 'alert.manage'])

  if (!canSee) {
    return <PermissionDenied permission="alert.read" title="Settings" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Administration</p>
          <h1 className="title">Settings</h1>
          <p className="lede">Control plane options for this deployment.</p>
        </div>
      </header>

      <div className="settings-layout">
        <nav className="settings-nav" aria-label="Settings sections">
          <NavLink to="/settings/notifications" className={settingsNavClass}>
            Notifications
          </NavLink>
        </nav>
        <div className="settings-panel">
          <Outlet context={ctx} />
        </div>
      </div>
    </div>
  )
}

/** Default child when visiting /settings */
export function SettingsIndexRedirect() {
  return <Navigate to="/settings/notifications" replace />
}
