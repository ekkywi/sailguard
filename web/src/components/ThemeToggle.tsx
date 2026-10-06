import { useEffect, useState } from 'react'
import { resolveTheme, toggleTheme, type Theme } from '../lib/theme'

function IconSun() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.75" />
      <path
        d="M12 2.5v2.25M12 19.25V21.5M4.5 12H2.25M21.75 12H19.5M5.64 5.64l1.59 1.59M16.77 16.77l1.59 1.59M16.77 7.23l1.59-1.59M5.64 18.36l1.59-1.59"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  )
}

function IconMoon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M18.5 14.2A7.25 7.25 0 0 1 9.8 5.5 7.5 7.5 0 1 0 18.5 14.2Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
    </svg>
  )
}

export function ThemeToggle({ className = '' }: { className?: string }) {
  const [theme, setThemeState] = useState<Theme>(() => resolveTheme())

  useEffect(() => {
    const onChange = () => setThemeState(resolveTheme())
    window.addEventListener('storage', onChange)
    return () => window.removeEventListener('storage', onChange)
  }, [])

  function onClick() {
    setThemeState(toggleTheme())
  }

  const next = theme === 'dark' ? 'light' : 'dark'

  return (
    <button
      type="button"
      className={`btn btn-icon ${className}`.trim()}
      onClick={onClick}
      aria-label={`Switch to ${next} mode`}
      title={`Switch to ${next} mode`}
    >
      {theme === 'dark' ? <IconSun /> : <IconMoon />}
    </button>
  )
}
