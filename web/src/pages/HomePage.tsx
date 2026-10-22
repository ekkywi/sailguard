import { useCallback, useEffect, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../components/PermissionDenied'
import {
  getDashboardSummary,
  getViolationsTimeseries,
} from '../features/dashboard/api'
import ViolationsChart from '../features/dashboard/ViolationsChart'
import type {
  DashboardSummary,
  ViolationsTimeseries,
  ViolationsTimeseriesRange,
} from '../features/dashboard/types'
import type { AppOutletContext } from '../layouts/AppLayout'
import { formatDateTime, formatRelativeTime } from '../lib/format'
import { hasPermission } from '../lib/permissions'
import { useDocumentTitle } from '../lib/useDocumentTitle'

function eventTypeLabel(t: string): string {
  if (t === 'violation_detected') return 'Detected'
  if (t === 'violation_blocked') return 'Blocked'
  return t
}

export default function HomePage() {
  useDocumentTitle('Overview')
  const { me, meLoading, meError } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'event.read')
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [summaryError, setSummaryError] = useState<string | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(true)
  const [tsRange, setTsRange] = useState<ViolationsTimeseriesRange>('24h')
  const [timeseries, setTimeseries] = useState<ViolationsTimeseries | null>(null)
  const [tsLoading, setTsLoading] = useState(true)
  const [tsError, setTsError] = useState<string | null>(null)

  const loadSummary = useCallback(() => {
    if (!canRead) {
      setSummaryLoading(false)
      return
    }
    setSummaryLoading(true)
    setSummaryError(null)
    getDashboardSummary()
      .then((data) => {
        setSummary(data)
        setSummaryError(null)
      })
      .catch((err: unknown) => {
        setSummary(null)
        setSummaryError(
          err instanceof Error ? err.message : 'failed to load dashboard',
        )
      })
      .finally(() => setSummaryLoading(false))
  }, [canRead])

  const loadTimeseries = useCallback(
    (range: ViolationsTimeseriesRange) => {
      if (!canRead) {
        setTsLoading(false)
        return
      }
      setTsLoading(true)
      setTsError(null)
      getViolationsTimeseries(range)
        .then((data) => {
          setTimeseries(data)
          setTsError(null)
        })
        .catch((err: unknown) => {
          setTimeseries(null)
          setTsError(
            err instanceof Error ? err.message : 'failed to load timeseries',
          )
        })
        .finally(() => setTsLoading(false))
    },
    [canRead],
  )

  useEffect(() => {
    loadSummary()
  }, [loadSummary])

  useEffect(() => {
    loadTimeseries(tsRange)
  }, [loadTimeseries, tsRange])

  const refreshAll = () => {
    loadSummary()
    loadTimeseries(tsRange)
  }

  const devices = summary?.devices
  const violations = summary?.violations_24h
  const recent = summary?.recent_violations ?? []
  const topProcesses = summary?.top_processes ?? []
  const topHosts = summary?.top_hosts ?? []
  const staleDevices = summary?.stale_devices ?? []

  if (!meLoading && !meError && me && !canRead) {
    return <PermissionDenied permission="event.read" title="Overview" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Overview</p>
          <h1 className="title">Control plane</h1>
          <p className="lede">
            Device counts and recent violations
            {summary ? (
              <span className="page-meta">
                {' '}
                · online ≤{summary.online_window_sec}s
              </span>
            ) : null}
          </p>
        </div>
        <button
          type="button"
          className="btn btn-ghost"
          onClick={refreshAll}
          disabled={summaryLoading || tsLoading}
        >
          Refresh
        </button>
      </header>

      {meError ? (
        <p className="error" role="alert">
          {meError}
        </p>
      ) : null}

      {summaryError ? (
        <p className="error" role="alert">
          {summaryError}
        </p>
      ) : null}

      {tsError ? (
        <p className="error" role="alert">
          {tsError}
        </p>
      ) : null}

      <div className="stat-grid" aria-label="Fleet summary">
        <Link to="/devices" className="stat-card">
          <span className="stat-label">Devices</span>
          <span className="stat-value">
            {devices ? devices.total : '—'}
          </span>
          <span className="stat-hint">
            {devices
              ? `${devices.active} active · ${devices.disabled} disabled`
              : 'View inventory →'}
          </span>
        </Link>
        <div className="stat-card stat-card-static">
          <span className="stat-label">Online</span>
          <span className="stat-value">
            {devices ? devices.online : '—'}
          </span>
          <span className="stat-hint">
            {devices
              ? `${devices.stale} stale (no recent heartbeat)`
              : 'Active + recent last_seen'}
          </span>
        </div>
        <Link to="/events" className="stat-card">
          <span className="stat-label">Violations 24h</span>
          <span className="stat-value">
            {violations
              ? violations.detected + violations.blocked
              : '—'}
          </span>
          <span className="stat-hint">
            {violations
              ? `${violations.detected} detected · ${violations.blocked} blocked`
              : 'View events →'}
          </span>
        </Link>
      </div>

      <ViolationsChart
        series={timeseries}
        loading={tsLoading}
        range={tsRange}
        onRangeChange={setTsRange}
      />

      <div className="dash-split" aria-label="Top offenders and stale devices">
        <section className="list-panel" aria-labelledby="dash-top-proc-heading">
          <div className="list-panel-header">
            <h2 id="dash-top-proc-heading" className="panel-title">
              Top processes (24h)
            </h2>
          </div>
          {summaryLoading && !summary ? (
            <p className="loading">Loading…</p>
          ) : null}
          {!summaryLoading && summary && topProcesses.length === 0 ? (
            <div className="empty-state" role="status">
              <p>No process hits in the last 24 hours.</p>
            </div>
          ) : null}
          {topProcesses.length > 0 ? (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Process</th>
                    <th scope="col">Count</th>
                  </tr>
                </thead>
                <tbody>
                  {topProcesses.map((p) => (
                    <tr key={p.process_name}>
                      <td>{p.process_name}</td>
                      <td>{p.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </section>

        <section className="list-panel" aria-labelledby="dash-top-host-heading">
          <div className="list-panel-header">
            <h2 id="dash-top-host-heading" className="panel-title">
              Top hosts (24h)
            </h2>
          </div>
          {summaryLoading && !summary ? (
            <p className="loading">Loading…</p>
          ) : null}
          {!summaryLoading && summary && topHosts.length === 0 ? (
            <div className="empty-state" role="status">
              <p>No host hits in the last 24 hours.</p>
            </div>
          ) : null}
          {topHosts.length > 0 ? (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Host</th>
                    <th scope="col">Count</th>
                  </tr>
                </thead>
                <tbody>
                  {topHosts.map((h) => (
                    <tr key={h.device_id}>
                      <td>
                        <Link to={`/devices/${h.device_id}`}>
                          {h.hostname || '—'}
                        </Link>
                      </td>
                      <td>{h.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </section>
      </div>

      <section className="list-panel" aria-labelledby="dash-stale-heading">
        <div className="list-panel-header">
          <h2 id="dash-stale-heading" className="panel-title">
            Stale devices
          </h2>
          <Link to="/devices" className="muted">
            All devices →
          </Link>
        </div>
        {summaryLoading && !summary ? (
          <p className="loading">Loading…</p>
        ) : null}
        {!summaryLoading && summary && staleDevices.length === 0 ? (
          <div className="empty-state" role="status">
            <p>All active devices have recent heartbeats.</p>
          </div>
        ) : null}
        {staleDevices.length > 0 ? (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Host</th>
                  <th scope="col">Last seen</th>
                </tr>
              </thead>
              <tbody>
                {staleDevices.map((d) => (
                  <tr key={d.device_id}>
                    <td>
                      <Link to={`/devices/${d.device_id}`}>
                        {d.hostname || '—'}
                      </Link>
                    </td>
                    <td>
                      {d.last_seen_at ? (
                        <span title={formatDateTime(d.last_seen_at)}>
                          {formatRelativeTime(d.last_seen_at)}
                        </span>
                      ) : (
                        'Never'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      <section className="list-panel" aria-labelledby="dash-recent-heading">
        <div className="list-panel-header">
          <h2 id="dash-recent-heading" className="panel-title">
            Recent violations
          </h2>
          <Link to="/events" className="muted">
            All events →
          </Link>
        </div>

        {summaryLoading && !summary ? (
          <p className="loading">Loading summary…</p>
        ) : null}

        {!summaryLoading && summary && recent.length === 0 ? (
          <div className="empty-state" role="status">
            <p>No violation events yet.</p>
          </div>
        ) : null}

        {recent.length > 0 ? (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">When</th>
                  <th scope="col">Host</th>
                  <th scope="col">Type</th>
                  <th scope="col">Process</th>
                </tr>
              </thead>
              <tbody>
                {recent.map((v) => (
                  <tr key={v.id}>
                    <td>
                      <span title={formatDateTime(v.occurred_at)}>
                        {formatRelativeTime(v.occurred_at)}
                      </span>
                    </td>
                    <td>
                      <Link to={`/devices/${v.device_id}`}>{v.hostname || '—'}</Link>
                    </td>
                    <td>{eventTypeLabel(v.event_type)}</td>
                    <td>{v.process_name || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      {meLoading && !me ? <p className="loading">Loading profile…</p> : null}

      {me ? (
        <div className="info-list" aria-label="Account details">
          <div className="info-row">
            <span className="info-key">Name</span>
            <span className="info-val">{me.name}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Email</span>
            <span className="info-val">{me.email}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Roles</span>
            <span className="info-val">
              {me.roles.length ? (
                <span className="badge-row">
                  {me.roles.map((role) => (
                    <span key={role} className="badge">
                      {role}
                    </span>
                  ))}
                </span>
              ) : (
                '—'
              )}
            </span>
          </div>
          <div className="info-row">
            <span className="info-key">Permissions</span>
            <span className="info-val muted">
              {me.permissions.length
                ? `${me.permissions.length} granted`
                : '—'}
            </span>
          </div>
        </div>
      ) : null}
    </div>
  )
}
