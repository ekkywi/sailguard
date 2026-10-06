const STORAGE_KEY = 'sg_theme'

export type Theme = 'light' | 'dark'

export function getStoredTheme(): Theme | null {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw === 'light' || raw === 'dark') return raw
  return null
}

export function systemTheme(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light'
}

export function resolveTheme(stored: Theme | null = getStoredTheme()): Theme {
  return stored ?? systemTheme()
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
}

export function setTheme(theme: Theme) {
  localStorage.setItem(STORAGE_KEY, theme)
  applyTheme(theme)
}

export function toggleTheme(): Theme {
  const next: Theme = resolveTheme() === 'dark' ? 'light' : 'dark'
  setTheme(next)
  return next
}

/** Call once at app start (and keep in sync with OS if user never chose). */
export function initTheme() {
  applyTheme(resolveTheme())
}
