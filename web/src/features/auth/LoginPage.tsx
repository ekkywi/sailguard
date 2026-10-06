import { FormEvent, useState } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import { getToken, setToken } from '../../lib/auth-storage';
import { login } from './api';

export default function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('admin@sailguard.local')
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
    <main className="page">
      <p className="eyebrow">SailGuard</p>
      <h1>Sign in</h1>
      <p className="lede">Use your control-plane admin account.</p>

      <form
        onSubmit={onSubmit}
        style={{ maxWidth: 360, display: 'grid', gap: '0.75rem' }}
      >
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="username"
            style={{ display: 'block', width: '100%', marginTop: 4 }}
          />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
            style={{ display: 'block', width: '100%', marginTop: 4 }}
          />
        </label>
        {error && <p style={{ color: '#b91c1c', margin: 0 }}>{error}</p>}
        <button type="submit" disabled={loading}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </main>
  )
}