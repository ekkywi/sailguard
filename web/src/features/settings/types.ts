export type AlertChannel = {
  id: string
  kind: string
  name: string
  enabled: boolean
  url: string
  created_at: string
  updated_at: string
}

export type AlertChannelListData = {
  items: AlertChannel[]
}

export type AlertDelivery = {
  id: string
  channel_id: string
  event_id: string
  status: string
  attempts: number
  max_attempts: number
  last_error?: string
  next_retry_at?: string
  delivered_at?: string
  created_at: string
  updated_at: string
}

export type AlertDeliveryListData = {
  items: AlertDelivery[]
}
