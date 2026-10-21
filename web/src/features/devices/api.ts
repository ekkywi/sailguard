import { apiFetch } from '../../lib/api'
import type {
  CreateDeviceOverrideInput,
  Device,
  DeviceListData,
  DeviceOverride,
  DeviceOverrideListData,
} from './types'

export function listDevices() {
  return apiFetch<DeviceListData>('/v1/devices')
}

export function getDevice(id: string) {
  return apiFetch<Device>(`/v1/devices/${id}`)
}

export function patchDevice(
  id: string,
  body: { display_name?: string; status?: string },
) {
  return apiFetch<Device>(`/v1/devices/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function deleteDevice(id: string) {
  return apiFetch<{ deleted: boolean }>(`/v1/devices/${id}`, {
    method: 'DELETE',
  })
}

export function listDeviceOverrides(
  deviceId: string,
  opts?: { includeExpired?: boolean },
) {
  const q = new URLSearchParams()
  if (opts?.includeExpired) q.set('include_expired', 'true')
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return apiFetch<DeviceOverrideListData>(
    `/v1/devices/${deviceId}/overrides${suffix}`,
  )
}

export function createDeviceOverride(
  deviceId: string,
  body: CreateDeviceOverrideInput,
) {
  return apiFetch<DeviceOverride>(`/v1/devices/${deviceId}/overrides`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function deleteDeviceOverride(deviceId: string, overrideId: string) {
  return apiFetch<{ deleted: boolean }>(
    `/v1/devices/${deviceId}/overrides/${overrideId}`,
    { method: 'DELETE' },
  )
}
