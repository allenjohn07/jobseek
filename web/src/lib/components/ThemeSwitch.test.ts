import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ThemeSwitch from './ThemeSwitch.svelte'
import { THEME_KEY } from '$lib/theme'

// A controllable stand-in for the OS dark-mode preference.
function fakeSystem(dark: boolean) {
  const listeners = new Set<() => void>()
  const query = {
    matches: dark,
    addEventListener: (_: string, fn: () => void) => listeners.add(fn),
    removeEventListener: (_: string, fn: () => void) => listeners.delete(fn),
  }
  vi.stubGlobal('matchMedia', () => query)
  return {
    flip(next: boolean) {
      query.matches = next
      listeners.forEach((fn) => fn())
    },
  }
}

const theme = () => document.documentElement.dataset.theme

beforeEach(() => {
  localStorage.clear()
  delete document.documentElement.dataset.theme
})
afterEach(() => vi.unstubAllGlobals())

describe('ThemeSwitch', () => {
  it('starts on System and marks it selected', () => {
    fakeSystem(false)
    render(ThemeSwitch)
    expect(screen.getByRole('button', { name: 'Match system theme' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('switches to the white (light) theme and remembers it', async () => {
    fakeSystem(true) // OS is dark, user picks light anyway
    render(ThemeSwitch)
    await userEvent.click(screen.getByRole('button', { name: 'Light theme' }))
    expect(theme()).toBe('light')
    expect(localStorage.getItem(THEME_KEY)).toBe('light')
    expect(screen.getByRole('button', { name: 'Light theme' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('switches to dark and remembers it', async () => {
    fakeSystem(false)
    render(ThemeSwitch)
    await userEvent.click(screen.getByRole('button', { name: 'Dark theme' }))
    expect(theme()).toBe('dark')
    expect(localStorage.getItem(THEME_KEY)).toBe('dark')
  })

  it('restores the saved choice on load', () => {
    fakeSystem(false)
    localStorage.setItem(THEME_KEY, 'dark')
    render(ThemeSwitch)
    expect(screen.getByRole('button', { name: 'Dark theme' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('System forgets the saved choice and follows the OS, including later changes', async () => {
    const os = fakeSystem(true)
    localStorage.setItem(THEME_KEY, 'light')
    render(ThemeSwitch)
    await userEvent.click(screen.getByRole('button', { name: 'Match system theme' }))
    expect(localStorage.getItem(THEME_KEY)).toBeNull()
    expect(theme()).toBe('dark')

    os.flip(false)
    expect(theme()).toBe('light')
  })

  it('ignores OS changes once a theme is picked', async () => {
    const os = fakeSystem(false)
    render(ThemeSwitch)
    await userEvent.click(screen.getByRole('button', { name: 'Light theme' }))
    os.flip(true)
    expect(theme()).toBe('light')
  })
})
