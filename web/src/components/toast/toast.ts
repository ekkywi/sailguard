export type ToastKind = 'success' | 'error' | 'info'

export type ToastItem = {
  id: string
  kind: ToastKind
  title: string
  description?: string
  duration: number
}

const MAX_TOASTS = 3
const DEFAULT_DURATION: Record<ToastKind, number> = {
  success: 3500,
  error: 5000,
  info: 4000,
}

type Listener = () => void

let toasts: ToastItem[] = []
const listeners = new Set<Listener>()

function emit() {
  for (const listener of listeners) {
    listener()
  }
}

function push(
  kind: ToastKind,
  title: string,
  description?: string,
  duration?: number,
) {
  const item: ToastItem = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    kind,
    title,
    description,
    duration: duration ?? DEFAULT_DURATION[kind],
  }
  toasts = [...toasts, item].slice(-MAX_TOASTS)
  emit()
  return item.id
}

export function getToasts(): ToastItem[] {
  return toasts
}

export function subscribeToasts(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function dismissToast(id: string) {
  const next = toasts.filter((t) => t.id !== id)
  if (next.length === toasts.length) return
  toasts = next
  emit()
}

export const toast = {
  success(title: string, description?: string, duration?: number) {
    return push('success', title, description, duration)
  },
  error(title: string, description?: string, duration?: number) {
    return push('error', title, description, duration)
  },
  info(title: string, description?: string, duration?: number) {
    return push('info', title, description, duration)
  },
  dismiss: dismissToast,
}
