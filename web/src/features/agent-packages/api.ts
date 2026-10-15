import { getToken } from '../../lib/auth-storage'
import { apiFetch } from '../../lib/api'
import type { AgentPackageMeta, DownloadWindowsPackageInput } from './types'

export function getWindowsAgentPackageMeta() {
  return apiFetch<AgentPackageMeta>('/v1/agent-packages/windows')
}

/** Downloads ZIP (binary response — not JSON envelope). */
export async function downloadWindowsAgentPackage(
  body: DownloadWindowsPackageInput,
): Promise<void> {
  const headers = new Headers({
    'Content-Type': 'application/json',
  })
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch('/v1/agent-packages/windows', {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    let message = `download failed (${res.status})`
    try {
      const errBody = (await res.json()) as {
        error?: { message?: string }
      }
      if (errBody.error?.message) {
        message = errBody.error.message
      }
    } catch {
      /* ignore */
    }
    throw new Error(message)
  }

  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sailguard-agent-${body.channel}.zip`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
