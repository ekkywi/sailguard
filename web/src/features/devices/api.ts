import { apiFetch } from '../../lib/api'
import type { Device, DeviceListData } from './types'

export function listDevices() {
  return apiFetch<DeviceListData>('/v1/devices')
}

export function getDevice(id: string) {
  return apiFetch<Device>(`/v1/devices/${id}`)
}
