// Light/dark theme. The choice is saved in localStorage; "system" (no saved value) follows the OS.
// index.html applies the same logic inline before first paint, so there's no flash of the wrong theme.
export type ThemeChoice = 'system' | 'light' | 'dark'
export type Theme = 'light' | 'dark'

export const THEME_KEY = 'jobseek-theme'

const systemDark = () =>
  typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: dark)') : null

export function loadChoice(): ThemeChoice {
  try {
    const v = localStorage.getItem(THEME_KEY)
    return v === 'light' || v === 'dark' ? v : 'system'
  } catch {
    return 'system' // storage can be unavailable (privacy modes)
  }
}

export function saveChoice(choice: ThemeChoice) {
  try {
    if (choice === 'system') localStorage.removeItem(THEME_KEY)
    else localStorage.setItem(THEME_KEY, choice)
  } catch {
    // Not fatal: the theme still applies for this visit.
  }
}

export function resolveTheme(choice: ThemeChoice): Theme {
  if (choice !== 'system') return choice
  return systemDark()?.matches ? 'dark' : 'light'
}

export function applyTheme(choice: ThemeChoice) {
  document.documentElement.dataset.theme = resolveTheme(choice)
}

// Calls onChange when the OS switches between light and dark. Returns an unsubscribe function.
export function watchSystemTheme(onChange: () => void): () => void {
  const m = systemDark()
  if (!m) return () => {}
  m.addEventListener('change', onChange)
  return () => m.removeEventListener('change', onChange)
}
