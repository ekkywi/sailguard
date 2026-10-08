import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { formatDateTime, formatRelativeTime, statusBadgeClass } from '../../lib/format'
import { getDevice } from './api'
import type { Device } from './types'

export default function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [device, setDevice] = useState<Device | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) {
      setError('missing device id')
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    getDevice(id)
      .then((d) => {
        if (!cancelled) {
          setDevice(d)
          setError(null)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setDevice(null)
          setError(err instanceof Error ? err.message : 'failed to load device')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  return (
    <div className="stack">
      <header>
        <p className="section-label">
          <Link to="/devices" className="crumb">
            Devices
          </Link>
          <span className="crumb-sep">/</span>
          Detail
        </p>
        <h1 className="title">{device?.hostname ?? 'Device'}</h1>
        <p className="lede">Enrollment and agent metadata for this endpoint.</p>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading ? <p className="loading">Loading device…</p> : null}

      {device ? (
        <div className="info-list" aria-label="Device details">
          <DetailRow label="Status">
            <span className={statusBadgeClass(device.status)}>{device.status}</span>
          </DetailRow>
          <DetailRow label="Hostname">{device.hostname}</DetailRow>
          <DetailRow label="Display name">{device.display_name || '—'}</DetailRow>
          <DetailRow label="OS">
            {device.os}
            {device.os_version ? ` ${device.os_version}` : ''}
          </DetailRow>
          <DetailRow label="Agent">{device.agent_version || '—'}</DetailRow>
          <DetailRow label="Machine GUID">
            <span className="mono">{device.machine_guid ?? '—'}</span>
          </DetailRow>
          <DetailRow label="Last seen">
            <span title={device.last_seen_at ?? undefined}>
              {formatRelativeTime(device.last_seen_at)}
              {device.last_seen_at ? (
                <span className="muted"> · {formatDateTime(device.last_seen_at)}</span>
              ) : null}
            </span>
          </DetailRow>
          <DetailRow label="Enrolled">
            <span title={device.enrolled_at ?? undefined}>
              {formatRelativeTime(device.enrolled_at)}
              {device.enrolled_at ? (
                <span className="muted"> · {formatDateTime(device.enrolled_at)}</span>
              ) : null}
            </span>
          </DetailRow>
          <DetailRow label="ID">
            <span className="mono">{device.id}</span>
          </DetailRow>
        </div>
      ) : null}
    </div>
  )
}

function DetailRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="info-row">
      <span className="info-key">{label}</span>
      <span className="info-val">{children}</span>
    </div>
  )
}
