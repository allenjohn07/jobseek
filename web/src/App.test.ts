import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.svelte'
import * as api from '$lib/api'

vi.mock('$lib/api')

beforeEach(() => {
  vi.mocked(api.getProposals).mockResolvedValue([])
  vi.mocked(api.getJobs).mockResolvedValue([])
  vi.mocked(api.getBaseResume).mockResolvedValue({
    id: 1,
    jobId: null,
    sessionId: '',
    updatedAt: '',
    content: {
      basics: { name: 'Allen', headline: '', email: '', phone: '', location: '', links: '' },
      summary: { id: 'summary', text: '' },
      experience: [],
      education: [],
      skills: [],
      projects: [],
    },
  })
})
afterEach(() => {
  location.hash = ''
})

describe('App tabs', () => {
  it('opens on the Jobs tab', async () => {
    render(App)
    expect(await screen.findByText(/No jobs yet/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Jobs' })).toHaveAttribute('aria-current', 'page')
  })

  it('switches to JobTweak and back, and follows the URL hash', async () => {
    const user = userEvent.setup()
    render(App)
    await screen.findByText(/No jobs yet/)

    await user.click(screen.getByRole('link', { name: 'JobTweak' }))
    expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
    expect(screen.queryByText(/No jobs yet/)).not.toBeInTheDocument()
    expect(location.hash).toBe('#/jobtweak')

    await user.click(screen.getByRole('link', { name: 'Jobs' }))
    expect(await screen.findByText(/No jobs yet/)).toBeInTheDocument()
  })

  it('opens directly on JobTweak when the hash says so', async () => {
    location.hash = '#/jobtweak'
    render(App)
    expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
  })
})
