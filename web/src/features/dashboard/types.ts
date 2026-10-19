export type DashboardDeviceCounts = {
  total: number
  active: number
  disabled: number
  pending: number
  online: number
  stale: number
}

export type DashboardViolations24h = {
  detected: number
  blocked: number
}

export type DashboardRecentViolation = {
  id: string
  device_id: string
  hostname: string
  event_type: string
  process_name: string | null
  occurred_at: string
}

export type DashboardTopProcess = {
  process_name: string
  count: number
}

export type DashboardTopHost = {
  device_id: string
  hostname: string
  count: number
}

export type DashboardStaleDevice = {
  device_id: string
  hostname: string
  last_seen_at: string | null
}

export type DashboardSummary = {
  generated_at: string
  online_window_sec: number
  devices: DashboardDeviceCounts
  violations_24h: DashboardViolations24h
  recent_violations: DashboardRecentViolation[]
  top_processes: DashboardTopProcess[]
  top_hosts: DashboardTopHost[]
  stale_devices: DashboardStaleDevice[]
}

export type ViolationsTimeseriesRange = '24h' | '7d'

export type ViolationsTimeseriesPoint = {
  t: string
  detected: number
  blocked: number
}

export type ViolationsTimeseries = {
  range: ViolationsTimeseriesRange | string
  bucket: 'hour' | 'day' | string
  from: string
  to: string
  points: ViolationsTimeseriesPoint[]
}
