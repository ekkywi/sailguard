/** Short relative time for tables (e.g. "2h ago"). Falls back to locale date. */
export function formatRelativeTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return '—'

  const diffSec = Math.round((Date.now() - then) / 1000)
  const abs = Math.abs(diffSec)
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

  if (abs < 60) return rtf.format(-diffSec, 'second')
  const diffMin = Math.round(diffSec / 60)
  if (Math.abs(diffMin) < 60) return rtf.format(-diffMin, 'minute')
  const diffHour = Math.round(diffMin / 60)
  if (Math.abs(diffHour) < 48) return rtf.format(-diffHour, 'hour')
  const diffDay = Math.round(diffHour / 24)
  if (Math.abs(diffDay) < 30) return rtf.format(-diffDay, 'day')

  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function statusBadgeClass(status: string): string {
  const key = status.trim().toLowerCase()
  if (key === 'active') return 'badge badge-status-active'
  if (key === 'disabled' || key === 'revoked') return 'badge badge-status-disabled'
  if (key === 'offline' || key === 'inactive') return 'badge badge-status-offline'
  return 'badge'
}

/** Matches control-plane dashboard DefaultOnlineWindowSec (120s). */
export const DEVICE_ONLINE_WINDOW_SEC = 120

export type DevicePresence = 'online' | 'stale' | 'unknown'

/** Connectivity from last_seen_at — separate from device status active|disabled. */
export function devicePresence(
  lastSeenAt: string | null | undefined,
  nowMs: number = Date.now(),
): DevicePresence {
  if (!lastSeenAt) return 'unknown'
  const then = new Date(lastSeenAt).getTime()
  if (Number.isNaN(then)) return 'unknown'
  const ageSec = (nowMs - then) / 1000
  return ageSec <= DEVICE_ONLINE_WINDOW_SEC ? 'online' : 'stale'
}

export function presenceLabel(p: DevicePresence): string {
  if (p === 'online') return 'Online'
  if (p === 'stale') return 'Stale'
  return 'Unknown'
}

export function presenceBadgeClass(p: DevicePresence): string {
  if (p === 'online') return 'badge badge-status-active'
  if (p === 'stale') return 'badge badge-status-offline'
  return 'badge'
}
