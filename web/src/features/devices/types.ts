export type Device = {
    id: string
    hostname: string
    display_name: string
    os: string
    os_version: string
    agent_version: string
    machine_guid: string | null
    status: string
    last_seen_at: string | null
    enrolled_at: string | null
    created_at: string
    updated_at: string
}

export type DeviceListData = {
  items: Device[]
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