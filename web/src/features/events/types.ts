export type DeviceEvent = {
  id: string
  device_id: string
  hostname: string
  client_event_id: string
  event_type: string
  occurred_at: string
  received_at: string
  dedupe_key: string | null

  process_pid: number | null
  process_name: string | null
  process_path: string | null
  process_hash_sha256: string | null
  process_publisher: string | null
  process_product_name: string | null
  process_user_name: string | null

  match_rule_id: string | null
  match_action: string | null
  match_rule_type: string | null
  match_value: string | null
  match_category_code: string | null

  policy_version: number | null
  policy_mode: string | null
}

export type EventTypeFilter = '' | 'violation_detected' | 'violation_blocked'

export type EventListQuery = {
  from?: string
  to?: string
  event_type?: EventTypeFilter
  device_id?: string
  q?: string
  limit?: number
  offset?: number
}

export type EventListData = {
  items: DeviceEvent[]
  total: number
  limit: number
  offset: number
}
