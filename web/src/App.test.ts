import { render, screen, waitFor, within } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.svelte'
import * as api from '$lib/api'
import type { Job } from '$lib/types'

vi.mock('$lib/api')

const job = (over: Partial<Job> = {}): Job => ({
  id: 1,
  company: 'Acme',
  title: 'Engineer',
  url: '',
  status: 'wishlist',
  appliedOn: '',
  notes: '',
  createdAt: '2026-01-01 00:00:00',
  ...over,
})

beforeEach(() => {
  vi.mocked(api.getJobs).mockResolvedValue([job(), job({ id: 2, company: 'Beta', title: 'Designer' })])
})

describe('App', () => {
  it('lists jobs from the API', async () => {
    render(App)
    expect(await screen.findByDisplayValue('Acme')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Beta')).toBeInTheDocument()
    expect(screen.getByText('2 jobs')).toBeInTheDocument()
  })

  it('shows the empty state', async () => {
    vi.mocked(api.getJobs).mockResolvedValue([])
    render(App)
    expect(await screen.findByText(/No jobs yet/)).toBeInTheDocument()
  })

  it('shows an error banner when loading fails', async () => {
    vi.mocked(api.getJobs).mockRejectedValue(new Error('boom'))
    render(App)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('inline edit sends the full job to updateJob', async () => {
    vi.mocked(api.updateJob).mockResolvedValue(job({ company: 'Acme2' }))
    const user = userEvent.setup()
    render(App)
    const cell = await screen.findByDisplayValue('Acme')
    await user.clear(cell)
    await user.type(cell, 'Acme2{Enter}')
    await waitFor(() => expect(api.updateJob).toHaveBeenCalled())
    const [id, body] = vi.mocked(api.updateJob).mock.calls[0]
    expect(id).toBe(1)
    expect(body).toMatchObject({ company: 'Acme2', title: 'Engineer', status: 'wishlist' })
  })

  it('rolls back an inline edit and shows the error when the API fails', async () => {
    vi.mocked(api.updateJob).mockRejectedValue(new Error('save failed'))
    const user = userEvent.setup()
    render(App)
    const cell = (await screen.findByDisplayValue('Acme')) as HTMLInputElement
    await user.clear(cell)
    await user.type(cell, 'Nope{Enter}')
    expect(await screen.findByRole('alert')).toHaveTextContent('save failed')
    await waitFor(() => expect(cell.value).toBe('Acme'))
  })

  it('deletes only after confirmation', async () => {
    vi.mocked(api.deleteJob).mockResolvedValue()
    const user = userEvent.setup()
    render(App)
    await screen.findByDisplayValue('Acme')
    await user.click(screen.getAllByRole('button', { name: /delete/i })[0])
    expect(api.deleteJob).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: /delete/i }))
    await waitFor(() => expect(api.deleteJob).toHaveBeenCalledWith(1))
    await waitFor(() => expect(screen.queryByDisplayValue('Acme')).not.toBeInTheDocument())
  })
})
