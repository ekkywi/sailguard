import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from '../../components/toast'
import {
  downloadWindowsAgentPackage,
  getWindowsAgentPackageMeta,
} from './api'
import type { AgentChannel, AgentPackageMeta } from './types'

function formatBytes(n: number): string {
  if (!n) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

export default function AgentDownloadPage() {
  const [meta, setMeta] = useState<AgentPackageMeta | null>(null)
  const [channelID, setChannelID] = useState('windows-legacy')
  const [serverURL, setServerURL] = useState('http://127.0.0.1:18080')
  const [enrollmentToken, setEnrollmentToken] = useState('')
  const [dataDir, setDataDir] = useState(
    'C:\\ProgramData\\SailGuard\\agent',
  )
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    getWindowsAgentPackageMeta()
      .then((m) => {
        setMeta(m)
        if (m.default_server_url) {
          setServerURL(m.default_server_url)
        }
        if (m.default_data_dir) {
          setDataDir(m.default_data_dir)
        }
        const preferred =
          m.channels.find((c) => c.id === 'windows-legacy' && c.binary_available) ??
          m.channels.find((c) => c.downloadable && c.binary_available) ??
          m.channels.find((c) => c.downloadable)
        if (preferred) {
          setChannelID(preferred.id)
        }
      })
      .catch((err: unknown) => {
        setMeta(null)
        setError(
          err instanceof Error ? err.message : 'failed to load package info',
        )
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const selected: AgentChannel | null = useMemo(() => {
    if (!meta) return null
    return meta.channels.find((c) => c.id === channelID) ?? null
  }, [meta, channelID])

  async function onDownload(e: FormEvent) {
    e.preventDefault()
    if (!selected?.downloadable || !selected.binary_available) {
      setError('selected channel binary is not available on the server')
      return
    }
    const url = serverURL.trim()
    const token = enrollmentToken.trim()
    if (!url || !token) {
      setError('server URL and enrollment token are required')
      return
    }

    setDownloading(true)
    setError(null)
    try {
      await downloadWindowsAgentPackage({
        channel: selected.id,
        server_url: url,
        enrollment_token: token,
        data_dir: dataDir.trim() || undefined,
      })
      toast.success('Download started', 'Check your browser downloads folder.')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'download failed'
      setError(msg)
      toast.error(msg)
    } finally {
      setDownloading(false)
    }
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Download agent</h1>
          <p className="lede">
            Choose an OS channel, then download an installer package. On the
            endpoint, run sailguard-agent-setup.exe as Administrator — it
            installs to Program Files and keeps data under ProgramData.
          </p>
        </div>
        <button
          type="button"
          className="btn btn-ghost"
          onClick={load}
          disabled={loading}
        >
          Refresh
        </button>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !meta ? <p className="loading">Loading package info…</p> : null}

      {meta ? (
        <section className="panel" aria-labelledby="channels-heading">
          <h2 id="channels-heading" className="panel-title">
            Channels
          </h2>
          <p className="muted" style={{ marginTop: 0 }}>
            Legacy Windows stays on Go 1.20; modern Windows tracks the current
            toolchain. Linux package slot is reserved.
          </p>
          <div className="channel-grid" role="radiogroup" aria-label="Agent channel">
            {meta.channels.map((ch) => {
              const selectedCh = ch.id === channelID
              const disabled = ch.coming_soon || !ch.downloadable
              return (
                <button
                  key={ch.id}
                  type="button"
                  role="radio"
                  aria-checked={selectedCh}
                  className={
                    selectedCh ? 'channel-card channel-card-active' : 'channel-card'
                  }
                  disabled={disabled}
                  onClick={() => setChannelID(ch.id)}
                >
                  <div className="channel-card-top">
                    <span className="channel-card-label">{ch.label}</span>
                    {ch.coming_soon ? (
                      <span className="badge">coming soon</span>
                    ) : ch.binary_available ? (
                      <span className="badge badge-status-active">available</span>
                    ) : (
                      <span className="badge badge-status-offline">missing</span>
                    )}
                  </div>
                  <p className="channel-card-desc">{ch.description}</p>
                  <p className="muted mono channel-card-meta">
                    {ch.toolchain}
                    {ch.binary_available
                      ? ` · ${formatBytes(ch.binary_size_bytes)}`
                      : ''}
                  </p>
                </button>
              )
            })}
          </div>
        </section>
      ) : null}

      {selected && !selected.coming_soon && !selected.binary_available ? (
        <div className="empty-state" role="status">
          <p className="empty-title">Build this channel first</p>
          <p className="muted mono">{selected.build_hint}</p>
          {selected.binary_path ? (
            <p className="muted mono">Expected path: {selected.binary_path}</p>
          ) : null}
        </div>
      ) : null}

      {meta && meta.setup_binary_available === false ? (
        <div className="empty-state" role="status">
          <p className="empty-title">Build the Windows setup first</p>
          <p className="muted mono">
            {meta.setup_build_hint ?? 'Run ./scripts/build-agent-setup.sh'}
          </p>
          {meta.setup_binary_path ? (
            <p className="muted mono">Expected path: {meta.setup_binary_path}</p>
          ) : null}
        </div>
      ) : null}

      <section className="panel" aria-labelledby="download-heading">
        <h2 id="download-heading" className="panel-title">
          Package settings
        </h2>
        <p className="muted" style={{ marginTop: 0 }}>
          Paste an enrollment token from <Link to="/tokens">Tokens</Link>{' '}
          (shown once at create). Selected channel:{' '}
          <span className="mono">{selected?.id ?? '—'}</span>
          {meta?.setup_binary_available ? (
            <>
              {' '}
              · setup{' '}
              <span className="mono">
                {formatBytes(meta.setup_binary_size_bytes ?? 0)}
              </span>
            </>
          ) : null}
        </p>
        <form className="token-create-grid" onSubmit={onDownload}>
          <div className="field" style={{ gridColumn: '1 / -1' }}>
            <label className="label" htmlFor="pkg-server">
              Server URL (agent → API)
            </label>
            <input
              id="pkg-server"
              className="input"
              value={serverURL}
              onChange={(e) => setServerURL(e.target.value)}
              placeholder="http://172.31.254.26:18080"
              required
            />
          </div>
          <div className="field" style={{ gridColumn: '1 / -1' }}>
            <label className="label" htmlFor="pkg-token">
              Enrollment token
            </label>
            <input
              id="pkg-token"
              className="input mono"
              value={enrollmentToken}
              onChange={(e) => setEnrollmentToken(e.target.value)}
              placeholder="paste token secret"
              required
              autoComplete="off"
            />
          </div>
          <div className="field" style={{ gridColumn: '1 / -1' }}>
            <label className="label" htmlFor="pkg-datadir">
              Data directory (on endpoint)
            </label>
            <input
              id="pkg-datadir"
              className="input mono"
              value={dataDir}
              onChange={(e) => setDataDir(e.target.value)}
              placeholder="C:\\ProgramData\\SailGuard\\agent"
            />
            <p className="muted" style={{ marginBottom: 0 }}>
              Setup always prefers ProgramData so reinstall keeps{' '}
              <span className="mono">device.json</span>.
            </p>
          </div>
          <div className="token-create-submit">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={
                downloading ||
                !selected?.downloadable ||
                !selected.binary_available ||
                meta?.setup_binary_available === false
              }
            >
              {downloading ? 'Preparing…' : 'Download installer package'}
            </button>
          </div>
        </form>
      </section>
    </div>
  )
}
