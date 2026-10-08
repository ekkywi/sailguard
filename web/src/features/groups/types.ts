export type DeviceGroup = {
    id: string
    name: string
    description: string
    created_at: string
    updated_at: string
}

export type GroupListData = {
    items: DeviceGroup[]
}

export type CreateGroupInput = {
    name: string
    description?: string
}