import { apiFetch } from '../../lib/api'
import type {
  CreateGroupInput,
  DeviceGroup,
  GroupListData,
  GroupMemberListData,
  GroupMembership,
} from './types'

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

export function listGroupMembers(groupId: string) {
  return apiFetch<GroupMemberListData>(`/v1/groups/${groupId}/members`)
}

export function addGroupMember(groupId: string, deviceId: string) {
  return apiFetch<GroupMembership>(`/v1/groups/${groupId}/members`, {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId }),
  })
}

export function removeGroupMember(groupId: string, deviceId: string) {
  return apiFetch<GroupMembership>(
    `/v1/groups/${groupId}/members/${deviceId}`,
    { method: 'DELETE' },
  )
}