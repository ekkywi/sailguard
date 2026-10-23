import { Navigate } from 'react-router-dom'
import { getToken } from '../../lib/auth-storage'
import { IdleSessionGuard } from './IdleSessionGuard'

export function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  return <IdleSessionGuard>{children}</IdleSessionGuard>
}
