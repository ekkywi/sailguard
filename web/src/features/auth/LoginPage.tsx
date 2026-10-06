import { FormEvent, useId, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { ThemeToggle } from '../../components/ThemeToggle'
import { getToken, setToken } from '../../lib/auth-storage'
import { login } from './api'

export default function LoginPage() {
  const navigate = useNavigate()
  const emailId = useId()
  const passwordId = useId()
  const errorId = useId()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (getToken()) {
    return <Navigate to="/" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const data = await login(email, password)
      setToken(data.access_token)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-screen">
      <div className="login-theme">
        <ThemeToggle />
      </div>

      <div className="login-panel">
        <h1 className="login-brand">SailGuard</h1>
        <p className="lede lede-tight">Sign in to the control plane</p>

        <form className="form" onSubmit={onSubmit} noValidate>
          <div className="field">
            <label className="label" htmlFor={emailId}>
              Email
            </label>
            <input
              id={emailId}
              className="input"
              type="email"
              name="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="username"
              placeholder="admin@example.com"
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errorId : undefined}
            />
          </div>

          <div className="field">
            <label className="label" htmlFor={passwordId}>
              Password
            </label>
            <input
              id={passwordId}
              className="input"
              type="password"
              name="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errorId : undefined}
            />
          </div>

          {error ? (
            <p id={errorId} className="error" role="alert">
              {error}
            </p>
          ) : null}

          <button
            type="submit"
            className="btn btn-primary btn-block"
            disabled={loading}
            aria-busy={loading}
          >
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>

      <p className="login-footer">Endpoint application control · on-premises</p>
    </div>
  )
}
