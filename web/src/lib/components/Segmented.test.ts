import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Segmented from './Segmented.svelte'

describe('Segmented', () => {
  it('renders link items as navigation with the current one marked', () => {
    render(Segmented, {
      label: 'Sections',
      value: 'b',
      items: [
        { id: 'a', label: 'Alpha', href: '#/a' },
        { id: 'b', label: 'Beta', href: '#/b' },
      ],
    })
    expect(screen.getByRole('link', { name: 'Beta' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Alpha' })).not.toHaveAttribute('aria-current')
    expect(screen.getByRole('link', { name: 'Alpha' })).toHaveAttribute('href', '#/a')
  })

  it('renders toggle buttons that report the picked option', async () => {
    const onChange = vi.fn()
    render(Segmented, {
      label: 'View',
      value: 'edit',
      onChange,
      items: [
        { id: 'preview', label: 'preview' },
        { id: 'edit', label: 'edit' },
      ],
    })
    expect(screen.getByRole('group', { name: 'View' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'edit' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'preview' })).toHaveAttribute('aria-pressed', 'false')

    await userEvent.click(screen.getByRole('button', { name: 'preview' }))
    expect(onChange).toHaveBeenCalledWith('preview')
  })
})
