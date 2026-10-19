import { apiFetch } from '../../lib/api'
import type {
  DashboardSummary,
  ViolationsTimeseries,
  ViolationsTimeseriesRange,
} from './types'

export function getDashboardSummary() {
  return apiFetch<DashboardSummary>('/v1/dashboard/summary')
}

export function getViolationsTimeseries(range: ViolationsTimeseriesRange = '24h') {
  const q = new URLSearchParams({ range })
  return apiFetch<ViolationsTimeseries>(
    `/v1/dashboard/violations-timeseries?${q.toString()}`,
  )
}
