import { render, screen } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'
import StatusBadge from './StatusBadge.svelte'
import { STATUSES } from '$lib/types'

describe('StatusBadge', () => {
  it.each(STATUSES)('renders %s as a coloured mark, not a coloured box', (status) => {
    const { container } = render(StatusBadge, { status })
    expect(container.querySelector('[data-mark]')).toHaveClass(`bg-mark-${status}`)
    expect(screen.getByText(status).className).not.toMatch(/\bbg-/)
  })
})
