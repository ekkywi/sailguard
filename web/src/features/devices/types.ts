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