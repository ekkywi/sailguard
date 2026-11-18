import type { DeviceEvent, EventTypeFilter } from './types'

/** Logical product domain — wire event_type strings stay unchanged. */
export type EventDomain = 'app' | 'network'

export type EventTypeDef = {
  /** Wire value for GET /v1/events?event_type= */
  id: Exclude<EventTypeFilter, ''>
  domain: EventDomain
  /** Chip + table Type column */
  label: string
  showUserColumn: boolean
  targetPrimary: (e: DeviceEvent) => string
  targetSecondary: (e: DeviceEvent) => string | null
}

function appTargetPrimary(e: DeviceEvent): string {
  return e.process_name || '—'
}

function appTargetSecondary(e: DeviceEvent): string | null {
  return e.process_path || null
}

function networkTargetPrimary(e: DeviceEvent): string {
  const host = e.network_dest_ip || e.network_dest_value
  if (!host) return '—'
  if (e.network_dest_port != null) return `${host}:${e.network_dest_port}`
  return host
}

function networkTargetSecondary(e: DeviceEvent): string | null {
  const bits: string[] = []
  if (e.network_protocol) bits.push(e.network_protocol)
  if (e.network_direction) bits.push(e.network_direction)
  if (
    e.network_dest_value &&
    e.network_dest_ip &&
    e.network_dest_value !== e.network_dest_ip
  ) {
    bits.push(e.network_dest_value)
  }
  return bits.length > 0 ? bits.join(' · ') : null
}

/** Known event types. Add a row here when a new module ships events. */
export const EVENT_TYPE_DEFS: EventTypeDef[] = [
  {
    id: 'violation_detected',
    domain: 'app',
    label: 'App detected',
    showUserColumn: true,
    targetPrimary: appTargetPrimary,
    targetSecondary: appTargetSecondary,
  },
  {
    id: 'violation_blocked',
    domain: 'app',
    label: 'App blocked',
    showUserColumn: true,
    targetPrimary: appTargetPrimary,
    targetSecondary: appTargetSecondary,
  },
  {
    id: 'network_blocked',
    domain: 'network',
    label: 'Network blocked',
    showUserColumn: false,
    targetPrimary: networkTargetPrimary,
    targetSecondary: networkTargetSecondary,
  },
]

const byId = new Map(EVENT_TYPE_DEFS.map((d) => [d.id, d]))

export const EVENT_TYPE_CHIPS: { id: EventTypeFilter; label: string }[] = [
  { id: '', label: 'All' },
  ...EVENT_TYPE_DEFS.map((d) => ({ id: d.id as EventTypeFilter, label: d.label })),
]

export function getEventTypeDef(eventType: string): EventTypeDef | undefined {
  return byId.get(eventType as Exclude<EventTypeFilter, ''>)
}

export function eventTypeLabel(eventType: string): string {
  return getEventTypeDef(eventType)?.label ?? eventType
}

export function eventTargetPrimary(e: DeviceEvent): string {
  return getEventTypeDef(e.event_type)?.targetPrimary(e) ?? e.process_name ?? '—'
}

export function eventTargetSecondary(e: DeviceEvent): string | null {
  return getEventTypeDef(e.event_type)?.targetSecondary(e) ?? e.process_path ?? null
}

export function eventUserDisplay(e: DeviceEvent): string {
  const def = getEventTypeDef(e.event_type)
  if (def && !def.showUserColumn) return '—'
  return e.process_user_name || '—'
}
