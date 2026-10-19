import { apiFetch } from '../../lib/api'
import type { DashboardSummary } from './types'

export function getDashboardSummary() {
  return apiFetch<DashboardSummary>('/v1/dashboard/summary')
}
