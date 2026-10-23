import { clearToken, getToken } from './auth-storage'

/** Best-effort server revoke, then clear local token and go to login. */
export async function endSession(opts?: { reason?: string }) {
  const token = getToken()
  if (token) {
    try {
      await fetch('/v1/auth/logout', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })
    } catch {
      // local clear still proceeds
    }
  }
  clearToken()
  const path = window.location.pathname
  if (!path.startsWith('/login')) {
    const q = opts?.reason ? `?reason=${encodeURIComponent(opts.reason)}` : ''
    window.location.assign(`/login${q}`)
  }
}
