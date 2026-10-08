import { useEffect, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { fetchMe } from '../features/auth/api'
import type { Me } from '../features/auth/types'

export type AppOutletContext = {
  me: Me | null
  meLoading: boolean
  meError: string | null
  reloadMe: () => void
}

export default function AppLayout() {
  const [me, setMe] = useState<Me | null>(null)
  const [meLoading, setMeLoading] = useState(true)
  const [meError, setMeError] = useState<string | null>(null)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let cancelled = false
    setMeLoading(true)
    fetchMe()
      .then((data) => {
        if (cancelled) return
        setMe(data)
        setMeError(null)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setMe(null)
        setMeError(err instanceof Error ? err.message : 'failed to load profile')
      })
      .finally(() => {
        if (!cancelled) setMeLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [tick])

  const ctx: AppOutletContext = {
    me,
    meLoading,
    meError,
    reloadMe: () => setTick((n) => n + 1),
  }

  return (
    <AppShell userLabel={me?.email ?? null}>
      <Outlet context={ctx} />
    </AppShell>
  )
}
