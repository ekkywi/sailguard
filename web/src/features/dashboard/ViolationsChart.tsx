import type { ViolationsTimeseries } from './types'

type Props = {
  series: ViolationsTimeseries | null
  loading: boolean
  range: '24h' | '7d'
  onRangeChange: (range: '24h' | '7d') => void
}

function formatBucketLabel(iso: string, bucket: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  if (bucket === 'day') {
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }
  return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

export default function ViolationsChart({
  series,
  loading,
  range,
  onRangeChange,
}: Props) {
  const points = series?.points ?? []
  const max = points.reduce(
    (m, p) => Math.max(m, p.detected + p.blocked),
    0,
  )
  const scale = max > 0 ? max : 1
  const total = points.reduce((s, p) => s + p.detected + p.blocked, 0)

  return (
    <section className="list-panel" aria-labelledby="dash-ts-heading">
      <div className="list-panel-header">
        <h2 id="dash-ts-heading" className="panel-title">
          Violations over time
        </h2>
        <div className="dash-range" role="group" aria-label="Time range">
          <button
            type="button"
            className={
              range === '24h' ? 'btn btn-ghost dash-range-active' : 'btn btn-ghost'
            }
            onClick={() => onRangeChange('24h')}
            disabled={loading}
          >
            24h
          </button>
          <button
            type="button"
            className={
              range === '7d' ? 'btn btn-ghost dash-range-active' : 'btn btn-ghost'
            }
            onClick={() => onRangeChange('7d')}
            disabled={loading}
          >
            7d
          </button>
        </div>
      </div>

      {loading && !series ? <p className="loading">Loading chart…</p> : null}

      {!loading && series && total === 0 ? (
        <div className="empty-state" role="status">
          <p>No violations in this range.</p>
        </div>
      ) : null}

      {points.length > 0 && total > 0 ? (
        <div className="ts-chart" role="img" aria-label="Violations stacked bar chart">
          <div className="ts-bars">
            {points.map((p, i) => {
              const detectedPct = (p.detected / scale) * 100
              const blockedPct = (p.blocked / scale) * 100
              const label = formatBucketLabel(p.t, series?.bucket ?? 'hour')
              const title = `${label}: ${p.detected} detected, ${p.blocked} blocked`
              const labelEvery =
                series?.bucket === 'day' ? 1 : Math.max(1, Math.ceil(points.length / 8))
              const showLabel = i % labelEvery === 0 || i === points.length - 1
              return (
                <div key={p.t} className="ts-col" title={title}>
                  <div className="ts-stack">
                    <div
                      className="ts-seg ts-seg-blocked"
                      style={{ height: `${blockedPct}%` }}
                    />
                    <div
                      className="ts-seg ts-seg-detected"
                      style={{ height: `${detectedPct}%` }}
                    />
                  </div>
                  <span className="ts-label">{showLabel ? label : ''}</span>
                </div>
              )
            })}
          </div>
          <div className="ts-legend">
            <span className="ts-legend-item">
              <span className="ts-swatch ts-seg-detected" /> Detected
            </span>
            <span className="ts-legend-item">
              <span className="ts-swatch ts-seg-blocked" /> Blocked
            </span>
            <span className="muted">
              {series?.bucket === 'day' ? 'per day' : 'per hour'} · max {max}
            </span>
          </div>
        </div>
      ) : null}
    </section>
  )
}
