import { useState } from 'react'
import type { SortDir } from './useClientList'

/** Sort state for server-paged lists (Devices, Events, Audit). */
export function useServerSort(
  defaultKey: string,
  defaultDir: SortDir = 'asc',
) {
  const [sortKey, setSortKey] = useState(defaultKey)
  const [sortDir, setSortDir] = useState<SortDir>(defaultDir)

  function toggleSort(key: string) {
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  return {
    sortKey,
    sortDir,
    toggleSort,
    sort: sortKey,
    order: sortDir,
  }
}
