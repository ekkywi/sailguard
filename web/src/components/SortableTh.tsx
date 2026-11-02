import type { SortDir } from '../lib/useClientList'

type SortableThProps = {
  label: string
  columnKey: string
  sortKey: string
  sortDir: SortDir
  onSort: (key: string) => void
  className?: string
}

export function SortableTh({
  label,
  columnKey,
  sortKey,
  sortDir,
  onSort,
  className,
}: SortableThProps) {
  const active = sortKey === columnKey
  const ariaSort = active ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'

  return (
    <th
      className={className ? `th-sortable ${className}` : 'th-sortable'}
      aria-sort={ariaSort}
      scope="col"
    >
      <button
        type="button"
        className={active ? 'th-sort-btn th-sort-btn-on' : 'th-sort-btn'}
        onClick={() => onSort(columnKey)}
      >
        <span>{label}</span>
        <span className="th-sort-indicator" aria-hidden="true">
          {active ? (sortDir === 'asc' ? '↑' : '↓') : '↕'}
        </span>
      </button>
    </th>
  )
}
