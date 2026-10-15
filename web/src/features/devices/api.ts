import { apiFetch } from '../../lib/api'
import type { Device, DeviceListData } from './types'

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
