import { render, screen } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'
import StatusBadge from './StatusBadge.svelte'
import { STATUSES } from '$lib/types'

describe('StatusBadge', () => {
  it.each(STATUSES)('renders %s with its tag token classes', (status) => {
    render(StatusBadge, { status })
    expect(screen.getByText(status)).toHaveClass(`bg-tag-${status}-bg`, `text-tag-${status}-fg`)
  })
})
