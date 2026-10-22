import { useEffect } from 'react'

const APP = 'SailGuard'

/** Browser tab title: "{page} · SailGuard". Empty page → "SailGuard". */
export function useDocumentTitle(page?: string | null) {
  useEffect(() => {
    const label = page?.trim()
    document.title = label ? `${label} · ${APP}` : APP
  }, [page])
}
