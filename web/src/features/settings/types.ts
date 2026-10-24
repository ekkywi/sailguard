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
