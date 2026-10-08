import { apiFetch } from '../../lib/api'
import type { CreateGroupInput, DeviceGroup, GroupListData } from './types'

export function listGroups() {
  return apiFetch<GroupListData>('/v1/groups')
}

export function createGroup(body: CreateGroupInput) {
  return apiFetch<DeviceGroup>('/v1/groups', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getGroup(id: string) {
  return apiFetch<DeviceGroup>(`/v1/groups/${id}`)
}