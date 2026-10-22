import { FormEvent, useId, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { SailGuardLogo } from '../../components/SailGuardLogo'
import { ThemeToggle } from '../../components/ThemeToggle'
import { getToken, setToken } from '../../lib/auth-storage'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { login } from './api'

function IconEye() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M2.5 12s3.5-6.5 9.5-6.5S21.5 12 21.5 12s-3.5 6.5-9.5 6.5S2.5 12 2.5 12Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="12" r="2.75" stroke="currentColor" strokeWidth="1.75" />
    </svg>
  )
}

function IconEyeOff() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M3.5 3.5 20.5 20.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <path
        d="M9.9 5.6C10.6 5.4 11.3 5.5 12 5.5c6 0 9.5 6.5 9.5 6.5a16 16 0 0 1-3.2 3.5M6.4 7.3A15.5 15.5 0 0 0 2.5 12S6 18.5 12 18.5c1.1 0 2.1-.2 3-.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M10.1 10.2a2.75 2.75 0 0 0 3.7 3.7"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

export default function LoginPage() {
  useDocumentTitle('Sign in')
  const navigate = useNavigate()
  const emailId = useId()
  const passwordId = useId()
  const errorId = useId()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [emailError, setEmailError] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (getToken()) {
    return <Navigate to="/" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setEmailError(null)
    setPasswordError(null)

    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      setEmailError('Email is required')
      return
    }
    if (!password) {
      setPasswordError('Password is required')
      return
    }

    setLoading(true)
    try {
      const data = await login(trimmedEmail, password)
      setToken(data.access_token)
      navigate('/', { replace: true })
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Sign-in failed'
      setError(msg)
      setPasswordError(msg)
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
        <div className="login-brand-block">
          <div className="login-brand-row">
            <SailGuardLogo className="sg-logo sg-logo-xl" />
            <h1 className="login-brand">SailGuard</h1>
          </div>
          <hr className="login-brand-rule" />
          <p className="lede lede-tight">Sign in</p>
        </div>

        <form className="form" onSubmit={onSubmit} noValidate>
          <div className="field">
            <label className="label" htmlFor={emailId}>
              Email
            </label>
            <input
              id={emailId}
              className={emailError ? 'input input-invalid' : 'input'}
              type="email"
              name="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                setEmailError(null)
                setError(null)
              }}
              required
              autoComplete="username"
              placeholder="admin@example.com"
              aria-invalid={emailError || error ? true : undefined}
              aria-describedby={emailError ? `${emailId}-err` : undefined}
            />
            {emailError ? (
              <p id={`${emailId}-err`} className="field-error" role="alert">
                {emailError}
              </p>
            ) : null}
          </div>

          <div className="field">
            <label className="label" htmlFor={passwordId}>
              Password
            </label>
            <div className="input-with-action">
              <input
                id={passwordId}
                className={passwordError ? 'input input-invalid' : 'input'}
                type={showPassword ? 'text' : 'password'}
                name="password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value)
                  setPasswordError(null)
                  setError(null)
                }}
                required
                autoComplete="current-password"
                aria-invalid={passwordError ? true : undefined}
                aria-describedby={passwordError ? errorId : undefined}
              />
              <button
                type="button"
                className="btn btn-icon input-action"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
                title={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <IconEyeOff /> : <IconEye />}
              </button>
            </div>
            {passwordError ? (
              <p id={errorId} className="field-error" role="alert">
                {passwordError}
              </p>
            ) : null}
          </div>

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

      <p className="login-footer">On-premises endpoint application control</p>
    </div>
  )
}
