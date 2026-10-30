export type Device = {
  id: string
  hostname: string
  display_name: string
  os: string
  os_version: string
  agent_version: string
  machine_guid: string | null
  status: string
  pic: string
  location: string
  asset_tag: string
  notes: string
  last_seen_at: string | null
  enrolled_at: string | null
  created_at: string
  updated_at: string
}

export type DeviceStatusFilter = '' | 'active' | 'disabled'
export type DeviceOSFilter = '' | 'windows' | 'linux' | 'darwin'

export type DeviceListQuery = {
  status?: string
  os?: string
  q?: string
  limit?: number
  offset?: number
}

export type DeviceListData = {
  items: Device[]
  total: number
  limit?: number
  offset?: number
}

export type DeviceOverride = {
  id: string
  device_id: string
  action: string
  rule_type: string
  operator: string
  value: string
  reason: string
  expires_at: string | null
  expired: boolean
  created_at: string
  updated_at: string
}

export type DeviceOverrideListData = {
  items: DeviceOverride[]
  include_expired: boolean
}

export type CreateDeviceOverrideInput = {
  action: string
  rule_type: string
  operator: string
  value: string
  reason: string
  expires_at?: string
}