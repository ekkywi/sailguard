import { useEffect, useMemo, useState } from 'react'

export type SortDir = 'asc' | 'desc'

export type ClientListCompare<T> = Record<string, (a: T, b: T) => number>

export function compareString(a: string, b: string): number {
  return a.localeCompare(b, undefined, { sensitivity: 'base' })
}

export function compareNumber(a: number, b: number): number {
  return a - b
}

export function compareDate(a: string | null | undefined, b: string | null | undefined): number {
  const ta = a ? Date.parse(a) : 0
  const tb = b ? Date.parse(b) : 0
  const na = Number.isFinite(ta) ? ta : 0
  const nb = Number.isFinite(tb) ? tb : 0
  return na - nb
}

type UseClientListOptions<T> = {
  items: T[] | null
  pageSize: number
  matches: (item: T, q: string) => boolean
  compare: ClientListCompare<T>
  defaultSortKey: string
  defaultSortDir?: SortDir
  filter?: (item: T) => boolean
  debounceMs?: number
}

export function useClientList<T>({
  items,
  pageSize,
  matches,
  compare,
  defaultSortKey,
  defaultSortDir = 'asc',
  filter,
  debounceMs = 250,
}: UseClientListOptions<T>) {
  const [queryDraft, setQueryDraft] = useState('')
  const [query, setQuery] = useState('')
  const [sortKey, setSortKey] = useState(defaultSortKey)
  const [sortDir, setSortDir] = useState<SortDir>(defaultSortDir)
  const [page, setPage] = useState(1)

  useEffect(() => {
    const t = window.setTimeout(() => {
      setQuery(queryDraft.trim())
      setPage(1)
    }, debounceMs)
    return () => window.clearTimeout(t)
  }, [queryDraft, debounceMs])

  const filtered = useMemo(() => {
    if (!items) return []
    const q = query.toLowerCase()
    return items.filter((item) => {
      if (filter && !filter(item)) return false
      if (!q) return true
      return matches(item, q)
    })
  }, [items, query, filter, matches])

  const sorted = useMemo(() => {
    const cmp = compare[sortKey]
    if (!cmp) return filtered
    const copy = [...filtered]
    copy.sort((a, b) => {
      const r = cmp(a, b)
      return sortDir === 'asc' ? r : -r
    })
    return copy
  }, [filtered, compare, sortKey, sortDir])

  const total = sorted.length
  const pageCount = Math.max(1, Math.ceil(total / pageSize) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    const start = (page - 1) * pageSize
    return sorted.slice(start, start + pageSize)
  }, [sorted, page, pageSize])

  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1
  const rangeEnd = Math.min(page * pageSize, total)

  const sourceTotal = items?.length ?? 0
  const hasActiveQuery = query.length > 0
  const isFiltered = Boolean(filter) || hasActiveQuery
  const filteredFromSource = isFiltered && sourceTotal !== total

  function toggleSort(key: string) {
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
    setPage(1)
  }

  function clearSearch() {
    setQueryDraft('')
    setQuery('')
    setPage(1)
  }

  return {
    queryDraft,
    setQueryDraft,
    query,
    sortKey,
    sortDir,
    toggleSort,
    page,
    setPage,
    pageCount,
    pageItems,
    total,
    rangeStart,
    rangeEnd,
    sourceTotal,
    filteredFromSource,
    clearSearch,
    hasActiveQuery,
  }
}
