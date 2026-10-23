import { useEffect } from 'react'
import { endSession } from '../../lib/session'

/** Idle timeout for admin console (layer 1). Default 10 minutes. */
const IDLE_MS = 10 * 60 * 1000

const ACTIVITY_EVENTS: (keyof WindowEventMap)[] = [
  'mousedown',
  'keydown',
  'mousemove',
  'scroll',
  'touchstart',
]

export function IdleSessionGuard({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    let timer = window.setTimeout(() => {
      void endSession({ reason: 'idle' })
    }, IDLE_MS)

    const bump = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(() => {
        void endSession({ reason: 'idle' })
      }, IDLE_MS)
    }

    for (const ev of ACTIVITY_EVENTS) {
      window.addEventListener(ev, bump, { passive: true })
    }
    return () => {
      window.clearTimeout(timer)
      for (const ev of ACTIVITY_EVENTS) {
        window.removeEventListener(ev, bump)
      }
    }
  }, [])

  return children
}
