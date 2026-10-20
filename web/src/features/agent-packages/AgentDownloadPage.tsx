import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { hasPermission } from '../../lib/permissions'
import {
  downloadWindowsAgentPackage,
  getWindowsAgentPackageMeta,
} from './api'
import type { AgentChannel, AgentPackageKind, AgentPackageMeta } from './types'

function formatBytes(n: number): string {
  if (!n) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function channelStatus(ch: AgentChannel): {
  label: string
  tone: 'ready' | 'missing' | 'soon'
} {
  if (ch.coming_soon) return { label: 'Coming soon', tone: 'soon' }
  if (ch.binary_available) return { label: 'Ready', tone: 'ready' }
  return { label: 'Not built', tone: 'missing' }
}

function PlatformGlyph({ family }: { family: string }) {
  if (family === 'linux') {
    return (
      <svg className="dl-glyph" viewBox="0 0 24 24" aria-hidden>
        <path
          fill="currentColor"
          d="M12 2c-1.4 1.2-2.2 3-2.2 4.8 0 .4.1.8.2 1.2C7.5 9 6 11.2 6 14c0 2.8 1.8 5 4.2 5.7-.3.5-.5 1.1-.5 1.8h4.6c0-.7-.2-1.3-.5-1.8C16.2 19 18 16.8 18 14c0-2.8-1.5-5-3.9-5.9.1-.4.2-.8.2-1.2C14.3 5 13.4 3.2 12 2z"
        />
      </svg>
    )
  }
  return (
    <svg className="dl-glyph" viewBox="0 0 24 24" aria-hidden>
      <path
        fill="currentColor"
        d="M3 5.5 11 4.3v7.2H3V5.5zm0 13L11 19.7v-7.2H3v6zm9 .2 9 1.3V12.5h-9v6.2zm0-14.4V11.5h9V3.6L12 4.3z"
      />
    </svg>
  )
}

export default function AgentDownloadPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canWrite = hasPermission(me, 'device.write')
  const [meta, setMeta] = useState<AgentPackageMeta | null>(null)
  const [channelID, setChannelID] = useState('windows-legacy')
  const [packageKind, setPackageKind] = useState<AgentPackageKind>('install')
  const [serverURL, setServerURL] = useState('http://127.0.0.1:18080')
  const [enrollmentToken, setEnrollmentToken] = useState('')
  const [dataDir, setDataDir] = useState(
    'C:\\ProgramData\\SailGuard\\agent',
  )
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState(false)

  const load = useCallback(() => {
    if (!canWrite) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    getWindowsAgentPackageMeta()
      .then((m) => {
        setMeta(m)
        if (m.default_server_url) setServerURL(m.default_server_url)
        if (m.default_data_dir) setDataDir(m.default_data_dir)
        const preferred =
          m.channels.find((c) => c.id === 'windows-legacy' && c.binary_available) ??
          m.channels.find((c) => c.downloadable && c.binary_available) ??
          m.channels.find((c) => c.downloadable)
        if (preferred) setChannelID(preferred.id)
      })
      .catch((err: unknown) => {
        setMeta(null)
        setError(
          err instanceof Error ? err.message : 'failed to load package info',
        )
      })
      .finally(() => setLoading(false))
  }, [canWrite])

  useEffect(() => {
    load()
  }, [load])

  const selected: AgentChannel | null = useMemo(() => {
    if (!meta) return null
    return meta.channels.find((c) => c.id === channelID) ?? null
  }, [meta, channelID])

  const canDownload = Boolean(
    selected?.downloadable &&
      selected.binary_available &&
      meta?.setup_binary_available !== false,
  )

  const blockingReason = useMemo(() => {
    if (!meta || !selected) return null
    if (selected.coming_soon || !selected.downloadable) {
      return 'This platform channel is not available yet.'
    }
    if (!selected.binary_available) {
      return selected.build_hint || 'Build the agent binary for this channel first.'
    }
    if (meta.setup_binary_available === false) {
      return meta.setup_build_hint ?? 'Run ./scripts/build-agent-setup.sh'
    }
    return null
  }, [meta, selected])

  const zipName = selected
    ? `sailguard-agent-${selected.id}-${packageKind}.zip`
    : '—'

  const contents = useMemo(() => {
    if (!selected) return []
    const rows = [
      {
        name: selected.binary_name,
        detail: selected.binary_available
          ? formatBytes(selected.binary_size_bytes)
          : 'missing',
      },
      {
        name: meta?.setup_binary_name ?? 'sailguard-agent-setup.exe',
        detail: meta?.setup_binary_available
          ? formatBytes(meta.setup_binary_size_bytes ?? 0)
          : 'missing',
      },
      {
        name: 'config.json',
        detail:
          packageKind === 'install' ? 'includes enrollment token' : 'server URL only',
      },
      { name: 'README.txt', detail: 'install steps' },
    ]
    return rows
  }, [selected, meta, packageKind])

  async function onDownload(e: FormEvent) {
    e.preventDefault()
    if (!selected?.downloadable || !selected.binary_available) {
      setError('selected channel binary is not available on the server')
      return
    }
    const url = serverURL.trim()
    const token = enrollmentToken.trim()
    if (!url) {
      setError('server URL is required')
      return
    }
    if (packageKind === 'install' && !token) {
      setError('enrollment token is required for first-install packages')
      return
    }

    setDownloading(true)
    setError(null)
    try {
      await downloadWindowsAgentPackage({
        channel: selected.id,
        package_kind: packageKind,
        server_url: url,
        enrollment_token: packageKind === 'install' ? token : undefined,
        data_dir: dataDir.trim() || undefined,
      })
      toast.success(
        'Download started',
        packageKind === 'update'
          ? 'Keep ProgramData device.json when installing the update.'
          : 'Check your browser downloads folder.',
      )
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'download failed'
      setError(msg)
      toast.error(msg)
    } finally {
      setDownloading(false)
    }
  }

  if (!canWrite) {
    return (
      <PermissionDenied permission="device.write" title="Download agent" />
    )
  }

  return (
    <div className="stack download-page">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Download agent</h1>
          <p className="lede">
            Package a Windows agent for first enrollment or a credential-safe
            exe update.
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

      {loading && !meta ? (
        <div className="download-shell" aria-busy="true">
          <div className="panel download-skeleton">
            <div className="dl-skel dl-skel-lg" />
            <div className="dl-skel" />
            <div className="dl-skel" />
            <div className="dl-skel dl-skel-md" />
          </div>
          <div className="panel download-skeleton">
            <div className="dl-skel dl-skel-md" />
            <div className="dl-skel" />
            <div className="dl-skel" />
          </div>
        </div>
      ) : null}

      {meta ? (
        <form className="download-shell" onSubmit={onDownload}>
          <div className="panel download-main">
            <section className="dl-block">
              <header className="dl-block-head">
                <p className="dl-kicker">Package type</p>
                <h2 className="dl-heading">What are you deploying?</h2>
              </header>
              <div className="dl-choice-grid" role="radiogroup" aria-label="Package type">
                <button
                  type="button"
                  role="radio"
                  aria-checked={packageKind === 'install'}
                  className={
                    packageKind === 'install'
                      ? 'dl-choice dl-choice-active'
                      : 'dl-choice'
                  }
                  onClick={() => setPackageKind('install')}
                >
                  <span className="dl-choice-radio" aria-hidden />
                  <span className="dl-choice-body">
                    <span className="dl-choice-title">First install</span>
                    <span className="dl-choice-desc">
                      New device. Requires an enrollment token (create one under
                      Tokens first).
                    </span>
                  </span>
                </button>
                <button
                  type="button"
                  role="radio"
                  aria-checked={packageKind === 'update'}
                  className={
                    packageKind === 'update'
                      ? 'dl-choice dl-choice-active'
                      : 'dl-choice'
                  }
                  onClick={() => setPackageKind('update')}
                >
                  <span className="dl-choice-radio" aria-hidden />
                  <span className="dl-choice-body">
                    <span className="dl-choice-title">Update</span>
                    <span className="dl-choice-desc">
                      Replace the binary only. Keeps{' '}
                      <span className="mono">device.json</span> in ProgramData.
                    </span>
                  </span>
                </button>
              </div>
            </section>

            <section className="dl-block">
              <header className="dl-block-head">
                <p className="dl-kicker">Platform</p>
                <h2 className="dl-heading">Choose a build channel</h2>
              </header>
              <div className="dl-platform-list" role="radiogroup" aria-label="Platform">
                {meta.channels.map((ch) => {
                  const active = ch.id === channelID
                  const disabled = ch.coming_soon || !ch.downloadable
                  const status = channelStatus(ch)
                  return (
                    <button
                      key={ch.id}
                      type="button"
                      role="radio"
                      aria-checked={active}
                      className={
                        active ? 'dl-platform dl-platform-active' : 'dl-platform'
                      }
                      disabled={disabled}
                      onClick={() => setChannelID(ch.id)}
                    >
                      <span className="dl-platform-icon">
                        <PlatformGlyph family={ch.os_family} />
                      </span>
                      <span className="dl-platform-body">
                        <span className="dl-platform-title-row">
                          <span className="dl-platform-title">{ch.label}</span>
                          <span
                            className={`dl-status dl-status-${status.tone}`}
                          >
                            {status.label}
                          </span>
                        </span>
                        <span className="dl-platform-meta muted">
                          {ch.toolchain}
                          {ch.binary_available
                            ? ` · ${formatBytes(ch.binary_size_bytes)}`
                            : ''}
                        </span>
                      </span>
                      <span
                        className={
                          active
                            ? 'dl-choice-radio dl-choice-radio-on'
                            : 'dl-choice-radio'
                        }
                        aria-hidden
                      />
                    </button>
                  )
                })}
              </div>
            </section>

            {blockingReason ? (
              <div className="callout callout-warn" role="status">
                <p className="callout-title">Build artifacts missing</p>
                <p className="callout-body">{blockingReason}</p>
                {selected?.binary_path && !selected.binary_available ? (
                  <p className="callout-meta mono">{selected.binary_path}</p>
                ) : null}
                {meta.setup_binary_available === false && meta.setup_binary_path ? (
                  <p className="callout-meta mono">{meta.setup_binary_path}</p>
                ) : null}
              </div>
            ) : null}

            {packageKind === 'update' && canDownload ? (
              <div className="callout callout-info" role="note">
                <p className="callout-title">Update checklist</p>
                <ol className="callout-steps">
                  <li>Run setup as Administrator (it stops/replaces/starts the service).</li>
                  <li>Leave ProgramData in place — do not delete device.json.</li>
                  <li>Confirm with <span className="mono">sc query SailGuardAgent</span>.</li>
                </ol>
              </div>
            ) : null}

            <section className="dl-block">
              <header className="dl-block-head">
                <p className="dl-kicker">Configuration</p>
                <h2 className="dl-heading">Endpoint settings</h2>
              </header>
              <div className="download-fields">
                <div className="field">
                  <label className="label" htmlFor="pkg-server">
                    Control plane URL
                  </label>
                  <input
                    id="pkg-server"
                    className="input"
                    value={serverURL}
                    onChange={(e) => setServerURL(e.target.value)}
                    placeholder="http://172.31.254.26:18080"
                    required
                  />
                  <p className="field-hint">
                    URL the agent uses to reach this API (LAN IP in lab
                    environments).
                  </p>
                </div>

                {packageKind === 'install' ? (
                  <div className="field">
                    <div className="label-row">
                      <label className="label" htmlFor="pkg-token">
                        Enrollment token
                      </label>
                      <Link className="crumb dl-inline-link" to="/tokens">
                        Manage tokens
                      </Link>
                    </div>
                    <input
                      id="pkg-token"
                      className="input mono"
                      value={enrollmentToken}
                      onChange={(e) => setEnrollmentToken(e.target.value)}
                      placeholder="Paste token secret"
                      required
                      autoComplete="off"
                    />
                    <p className="field-hint">
                      Embedded in <span className="mono">config.json</span> for
                      first-time enrollment only.
                    </p>
                  </div>
                ) : null}

                <div className="download-advanced">
                  <button
                    type="button"
                    className="dl-advanced-toggle"
                    aria-expanded={showAdvanced}
                    onClick={() => setShowAdvanced((v) => !v)}
                  >
                    <span>{showAdvanced ? 'Hide' : 'Show'} advanced</span>
                    <span className="muted">Data directory override</span>
                  </button>
                  {showAdvanced ? (
                    <div className="field">
                      <label className="label" htmlFor="pkg-datadir">
                        Data directory
                      </label>
                      <input
                        id="pkg-datadir"
                        className="input mono"
                        value={dataDir}
                        onChange={(e) => setDataDir(e.target.value)}
                        placeholder="C:\\ProgramData\\SailGuard\\agent"
                      />
                      <p className="field-hint">
                        Default ProgramData path preserves credentials across
                        reinstalls.
                      </p>
                    </div>
                  ) : null}
                </div>
              </div>
            </section>
          </div>

          <aside className="panel download-aside" aria-label="Package summary">
            <p className="dl-kicker">Package summary</p>
            <h2 className="dl-aside-title">
              {packageKind === 'install' ? 'Installer ZIP' : 'Update ZIP'}
            </h2>
            <p className="dl-aside-file mono">{zipName}</p>

            <dl className="dl-meta-list">
              <div className="dl-meta-row">
                <dt>Platform</dt>
                <dd>{selected?.label ?? '—'}</dd>
              </div>
              <div className="dl-meta-row">
                <dt>Type</dt>
                <dd>
                  {packageKind === 'install' ? 'First install' : 'Update'}
                </dd>
              </div>
              <div className="dl-meta-row">
                <dt>Status</dt>
                <dd>
                  {canDownload ? (
                    <span className="dl-status dl-status-ready">Ready</span>
                  ) : (
                    <span className="dl-status dl-status-missing">Blocked</span>
                  )}
                </dd>
              </div>
            </dl>

            <div className="dl-contents">
              <p className="dl-contents-label">Includes</p>
              <ul className="dl-contents-list">
                {contents.map((item) => (
                  <li key={item.name}>
                    <span className="mono">{item.name}</span>
                    <span className="muted">{item.detail}</span>
                  </li>
                ))}
              </ul>
            </div>

            <button
              type="submit"
              className="btn btn-primary dl-aside-cta"
              disabled={downloading || !canDownload}
            >
              {downloading
                ? 'Preparing package…'
                : packageKind === 'update'
                  ? 'Download update'
                  : 'Download installer'}
            </button>
            <p className="field-hint dl-aside-hint">
              After download, run{' '}
              <span className="mono">sailguard-agent-setup.exe</span> as
              Administrator. Setup registers the{' '}
              <span className="mono">SailGuardAgent</span> service for auto-start.
            </p>
          </aside>
        </form>
      ) : null}
    </div>
  )
}
