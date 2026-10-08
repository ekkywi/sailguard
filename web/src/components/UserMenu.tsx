import { useEffect, useId, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { clearToken } from '../lib/auth-storage'

type UserMenuProps = {
  userLabel?: string | null
}

export function UserMenu({ userLabel }: UserMenuProps) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  const label = userLabel?.trim() || 'Account'
  const initial = (userLabel?.trim()?.[0] ?? 'A').toUpperCase()

  useEffect(() => {
    if (!open) return

    function onPointerDown(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }

    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  function logout() {
    setOpen(false)
    clearToken()
    navigate('/login', { replace: true })
  }

  return (
    <div className="user-menu" ref={rootRef}>
      <button
        type="button"
        className="user-menu-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="user-avatar" aria-hidden="true">
          {initial}
        </span>
        <span className="user-menu-label">{label}</span>
        <span className="user-menu-caret" aria-hidden="true">
          ▾
        </span>
      </button>

      {open ? (
        <div className="user-menu-panel" id={menuId} role="menu">
          <button
            type="button"
            role="menuitem"
            className="user-menu-item"
            onClick={logout}
          >
            Sign out
          </button>
        </div>
      ) : null}
    </div>
  )
}
