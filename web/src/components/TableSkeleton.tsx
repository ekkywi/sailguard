type TableSkeletonProps = {
  label?: string
  rows?: number
}

/** Shared list loading placeholder (devices-style shimmer). */
export function TableSkeleton({
  label = 'Loading…',
  rows = 3,
}: TableSkeletonProps) {
  return (
    <div className="loading-block" role="status" aria-busy="true">
      <div className="skeleton-table">
        <div className="skeleton-row skeleton-head" />
        {Array.from({ length: rows }, (_, i) => (
          <div key={i} className="skeleton-row" />
        ))}
      </div>
      <p className="loading skeleton-label">{label}</p>
    </div>
  )
}
