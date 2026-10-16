import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import {
  dismissToast,
  getToasts,
  subscribeToasts,
  type ToastItem,
  type ToastKind,
} from './toast'

function IconSuccess() {
  return (
    <svg className="toast-icon" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle cx="12" cy="12" r="8.25" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M8.5 12.2 10.8 14.5 15.5 9.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconError() {
  return (
    <svg className="toast-icon" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle cx="12" cy="12" r="8.25" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M9.5 9.5l5 5M14.5 9.5l-5 5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function IconInfo() {
  return (
    <svg className="toast-icon" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle cx="12" cy="12" r="8.25" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M12 11v5.5M12 8.25h.01"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function kindIcon(kind: ToastKind) {
  switch (kind) {
    case 'success':
      return <IconSuccess />
    case 'error':
      return <IconError />
    default:
      return <IconInfo />
  }
}

function ToastCard({ item }: { item: ToastItem }) {
  const [leaving, setLeaving] = useState(false)
  const [paused, setPaused] = useState(false)
  const remainingRef = useRef(item.duration)
  const startedAtRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  function clearTimer() {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  function schedule(ms: number) {
    clearTimer()
    startedAtRef.current = Date.now()
    remainingRef.current = ms
    timerRef.current = setTimeout(() => {
      setLeaving(true)
      window.setTimeout(() => dismissToast(item.id), 220)
    }, ms)
  }

  useEffect(() => {
    schedule(item.duration)
    return clearTimer
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only on mount per toast id
  }, [item.id, item.duration])

  function onEnter() {
    if (leaving) return
    setPaused(true)
    const elapsed = Date.now() - startedAtRef.current
    remainingRef.current = Math.max(0, remainingRef.current - elapsed)
    clearTimer()
  }

  function onLeave() {
    if (leaving) return
    setPaused(false)
    schedule(remainingRef.current || 800)
  }

  function onDismiss() {
    clearTimer()
    setLeaving(true)
    window.setTimeout(() => dismissToast(item.id), 220)
  }

  const role = item.kind === 'error' ? 'alert' : 'status'

  return (
    <div
      className={`toast-card toast-${item.kind}${leaving ? ' toast-leaving' : ''}`}
      role={role}
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
    >
      <div className="toast-body">
        <span className="toast-icon-wrap">{kindIcon(item.kind)}</span>
        <div className="toast-text">
          <p className="toast-title">{item.title}</p>
          {item.description ? (
            <p className="toast-desc">{item.description}</p>
          ) : null}
        </div>
        <button
          type="button"
          className="toast-close"
          aria-label="Dismiss"
          onClick={onDismiss}
        >
          ×
        </button>
      </div>
      <div
        className={`toast-progress${paused ? ' toast-progress-paused' : ''}`}
        style={{ animationDuration: `${item.duration}ms` }}
      />
    </div>
  )
}

export function ToastViewport() {
  const items = useSyncExternalStore(subscribeToasts, getToasts, getToasts)

  if (items.length === 0) return null

  return (
    <div className="toast-viewport" aria-live="polite" aria-relevant="additions">
      {items.map((item) => (
        <ToastCard key={item.id} item={item} />
      ))}
    </div>
  )
}
