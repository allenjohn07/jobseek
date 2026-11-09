import type { Editor } from '@tiptap/core'
import { render, screen, waitFor, within, act } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import JobTweakView from './JobTweakView.svelte'
import * as api from '$lib/api'
import type { Job, Proposal, Resume } from '$lib/types'

vi.mock('$lib/api')

const job = (over: Partial<Job> = {}): Job => ({
  id: 7,
  company: 'Acme',
  title: 'Engineer',
  url: '',
  status: 'wishlist',
  appliedOn: '',
  notes: '',
  description: 'Build Go services',
  createdAt: '2026-01-01 00:00:00',
  ...over,
})

const resume = (over: Partial<Resume> = {}): Resume => ({
  id: 1,
  jobId: null,
  sessionId: '',
  updatedAt: '',
  content: {
    basics: { name: 'Allen', headline: '', email: 'a@b.co', phone: '', location: '', links: '' },
    summary: { id: 'summary', text: 'Backend dev' },
    experience: [
      { id: 'e1', company: 'Acme', role: 'Dev', start: '2023', end: 'Present', bullets: [{ id: 'b1', text: 'Built APIs' }] },
    ],
    education: [],
    skills: [{ id: 's1', text: 'Go, SQL' }],
    projects: [],
  },
  ...over,
})

const proposal = (over: Partial<Proposal> = {}): Proposal => ({
  id: 11,
  resumeId: 1,
  op: 'replace',
  itemId: 'b1',
  beforeText: 'Built APIs',
  afterText: 'Built Go APIs',
  reason: 'Job asks for Go',
  status: 'pending',
  createdAt: '',
  ...over,
})

beforeEach(() => {
  vi.mocked(api.getProposals).mockResolvedValue([])
  vi.mocked(api.getJobs).mockResolvedValue([job()])
  vi.mocked(api.getBaseResume).mockResolvedValue(resume())
})

async function renderEditor() {
  render(JobTweakView)
  await userEvent.click(await screen.findByRole('button', { name: 'edit' }))
  await userEvent.click(screen.getByRole('button', { name: /Ask AI/ }))
}

describe('JobTweakView', () => {
  it('shows the base resume first and lists jobs to tailor for', async () => {
    await renderEditor()
    expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Base resume/ })).toHaveAttribute('aria-current', 'true')
    expect(screen.getByRole('button', { name: /Engineer/ })).toBeInTheDocument()
  })

  it('offers to create a tailored resume when the job has none, then shows the copy', async () => {
    vi.mocked(api.getJobResume).mockResolvedValue(null)
    vi.mocked(api.createJobResume).mockResolvedValue(resume({ id: 2, jobId: 7 }))
    const user = userEvent.setup()
    await renderEditor()
    await screen.findByRole('heading', { name: 'Allen' })

    await user.click(screen.getByRole('button', { name: /Engineer/ }))
    await user.click(await screen.findByRole('button', { name: /Create from base resume/ }))

    await waitFor(() => expect(api.createJobResume).toHaveBeenCalledWith(7))
    expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
    // The job description is editable next to the resume for a job.
    expect(screen.getByLabelText('Job description')).toHaveValue('Build Go services')
  })

  it('saves document edits with stable item IDs and the complete document', async () => {
    vi.mocked(api.updateResume).mockImplementation(async (id, content) => resume({ id, content }))
    await renderEditor()
    const element = screen.getByRole('textbox', { name: 'Resume document' }) as HTMLElement & { editor: Editor }
    await act(() => {
      element.editor.state.doc.descendants((node, pos) => {
        if (node.attrs.resumeKey === 'item:b1') { element.editor.commands.insertContentAt({ from: pos + 1, to: pos + 1 + node.content.size }, 'Built Go APIs'); return false }
      })
    })
    await waitFor(() => expect(api.updateResume).toHaveBeenCalled())
    const [id, content] = vi.mocked(api.updateResume).mock.calls[0]
    expect(id).toBe(1)
    expect(JSON.stringify(content.document)).toContain('Built Go APIs')
    expect(JSON.stringify(content.document)).toContain('item:b1')
    expect(content.summary.text).toBe('Backend dev')
  })

  it('keeps unsaved document edits available when saving fails', async () => {
    vi.mocked(api.updateResume).mockRejectedValue(new Error('save failed'))
    await renderEditor()
    const element = screen.getByRole('textbox', { name: 'Resume document' }) as HTMLElement & { editor: Editor }
    await act(() => { element.editor.commands.insertContentAt(1, 'Updated ') })
    await waitFor(() => expect(screen.getAllByRole('alert').some(node => node.textContent?.includes('save failed'))).toBe(true))
    expect(screen.getByRole('heading', { name: 'Updated Allen' })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Retry save' })).toBeInTheDocument()
    vi.mocked(api.updateResume).mockImplementation(async (id, content) => resume({ id, content }))
    await userEvent.click(screen.getByRole('button', { name: 'Retry save' }))
    expect(await screen.findByText('All changes saved')).toBeInTheDocument()
  })

  it('allows adding content anywhere in the document and undoing it', async () => {
    vi.mocked(api.updateResume).mockImplementation(async (id, content) => resume({ id, content }))
    await renderEditor()
    const element = screen.getByRole('textbox', { name: 'Resume document' }) as HTMLElement & { editor: Editor }
    await act(() => { element.editor.commands.insertContentAt(element.editor.state.doc.content.size, { type: 'paragraph', content: [{ type: 'text', text: 'Additional accomplishments' }] }) })
    expect(await screen.findByText('Additional accomplishments')).toBeInTheDocument()
    await waitFor(() => expect(api.updateResume).toHaveBeenCalled())
    await act(() => { element.editor.commands.undo() })
    await waitFor(() => expect(screen.queryByText('Additional accomplishments')).not.toBeInTheDocument())
  })

  it('previews the formatted resume and prints from there', async () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    const user = userEvent.setup()
    await renderEditor()
    await screen.findByRole('heading', { name: 'Allen' })
    expect(screen.getByRole('button', { name: /Export PDF/ })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'preview' }))
    const doc = document.querySelector('.resume-print') as HTMLElement
    expect(within(doc).getByRole('heading', { name: 'Allen' })).toBeInTheDocument()
    expect(within(doc).getByText('Built APIs')).toBeInTheDocument()
    expect(within(doc).getByText('Dev')).toBeInTheDocument()
    expect(within(doc).getByText('Acme')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Export PDF/ }))
    expect(print).toHaveBeenCalled()
  })

  it('shows an error banner when loading fails', async () => {
    vi.mocked(api.getBaseResume).mockRejectedValue(new Error('boom'))
    render(JobTweakView)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  describe('AI suggestions', () => {
    it('lists pending suggestions with the reason, and hides resolved ones', async () => {
      vi.mocked(api.getProposals).mockResolvedValue([
        proposal(),
        proposal({ id: 12, status: 'rejected', afterText: 'Old idea' }),
      ])
      await renderEditor()
      const panel = await screen.findByLabelText('AI suggestions')
      expect(within(panel).getByText('Built Go APIs')).toBeInTheDocument()
      expect(within(panel).getByText('Job asks for Go')).toBeInTheDocument()
      expect(within(panel).queryByText('Old idea')).not.toBeInTheDocument()
      expect(screen.getByText('AI suggestions (1)')).toBeInTheDocument()
    })

    it('accepting applies it on the server and shows the updated resume', async () => {
      vi.mocked(api.getProposals).mockResolvedValueOnce([proposal()]).mockResolvedValue([proposal({ status: 'accepted' })])
      vi.mocked(api.acceptProposal).mockResolvedValue(proposal({ status: 'accepted' }))
      const updated = resume()
      updated.content.experience[0].bullets[0].text = 'Built Go APIs'
      vi.mocked(api.getBaseResume).mockResolvedValueOnce(resume()).mockResolvedValue(updated)
      const user = userEvent.setup()
      await renderEditor()

      await user.click(await screen.findByRole('button', { name: 'Accept' }))
      await waitFor(() => expect(api.acceptProposal).toHaveBeenCalledWith(11))
      expect(await screen.findByText('Built Go APIs')).toBeInTheDocument()
      await waitFor(() => expect(screen.queryByLabelText('AI suggestions')).not.toBeInTheDocument())
    })

    it('rejecting leaves the resume alone', async () => {
      vi.mocked(api.getProposals).mockResolvedValueOnce([proposal()]).mockResolvedValue([proposal({ status: 'rejected' })])
      vi.mocked(api.rejectProposal).mockResolvedValue(proposal({ status: 'rejected' }))
      const user = userEvent.setup()
      await renderEditor()

      await user.click(await screen.findByRole('button', { name: 'Reject' }))
      await waitFor(() => expect(api.rejectProposal).toHaveBeenCalledWith(11))
      await waitFor(() => expect(screen.queryByLabelText('AI suggestions')).not.toBeInTheDocument())
      expect(api.acceptProposal).not.toHaveBeenCalled()
      expect(screen.getByText('Built APIs')).toBeInTheDocument()
    })

    it('shows the server message when a stale suggestion cannot be applied', async () => {
      vi.mocked(api.getProposals).mockResolvedValue([proposal()])
      vi.mocked(api.acceptProposal).mockRejectedValue(new Error('conflict: item changed since this was suggested'))
      const user = userEvent.setup()
      await renderEditor()

      await user.click(await screen.findByRole('button', { name: 'Accept' }))
      expect(await screen.findByRole('alert')).toHaveTextContent('item changed')
      // It stays pending so the user can still reject it.
      expect(screen.getByRole('button', { name: 'Reject' })).toBeInTheDocument()
    })

    it('shows the connect command', async () => {
      vi.mocked(api.getMcpInfo).mockResolvedValue({
        url: 'http://localhost:8080/mcp',
        token: 'abc',
        command: 'claude mcp add --transport http jobseek http://localhost:8080/mcp --header "Authorization: Bearer abc"',
      })
      const user = userEvent.setup()
      await renderEditor()
      await screen.findByRole('heading', { name: 'Allen' })
      await user.click(screen.getByRole('button', { name: /Connect AI/ }))
      expect(await screen.findByLabelText('Connect command')).toHaveTextContent('claude mcp add --transport http jobseek')
    })
  })

  describe('chat', () => {
    it('toggles follow-up chat in either view', async () => {
      const user = userEvent.setup()
      await renderEditor()
      expect(await screen.findByLabelText('AI chat')).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: 'Close assistant' }))
      expect(screen.getByLabelText('AI chat')).not.toBeVisible()
      await user.click(screen.getByRole('button', { name: /Ask AI/ }))
      expect(screen.getByLabelText('AI chat')).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: 'preview' }))
      expect(screen.getByLabelText('AI chat')).toBeInTheDocument()
    })

    it('refreshes suggestions when the AI proposes an edit', async () => {
      vi.mocked(api.streamChat).mockImplementation(async (_i, _m, onEvent) => {
        // The server stored a suggestion while the AI worked.
        vi.mocked(api.getProposals).mockResolvedValue([proposal()])
        onEvent({ type: 'tool', name: 'propose_edit' })
      })
      const user = userEvent.setup()
      await renderEditor()
      await screen.findByRole('heading', { name: 'Allen' })
      expect(screen.queryByLabelText('AI suggestions')).not.toBeInTheDocument()

      await user.type(screen.getByLabelText('Message'), 'tailor it{Enter}')
      expect(await screen.findByLabelText('AI suggestions')).toBeInTheDocument()
      expect(api.streamChat).toHaveBeenCalledWith(1, 'tailor it', expect.any(Function), expect.any(AbortSignal))
    })
  })
})

it('creates a job copy and tailors it in one click using the saved description', async () => {
  vi.mocked(api.getJobResume).mockResolvedValue(null)
  vi.mocked(api.createJobResume).mockResolvedValue(resume({ id: 2, jobId: 7 }))
  vi.mocked(api.streamChat).mockImplementation(async (_id, _message, onEvent) => {
    onEvent({ type: 'tool', name: 'propose_edit' })
  })
  const user = userEvent.setup()
  render(JobTweakView)
  await screen.findByRole('heading', { name: 'Allen' })
  await user.click(screen.getByRole('button', { name: /Engineer/ }))
  expect(await screen.findByLabelText('Job description')).toHaveValue('Build Go services')
  await user.click(screen.getByRole('button', { name: 'Tailor resume' }))
  await waitFor(() => expect(api.streamChat).toHaveBeenCalledWith(2, expect.stringContaining('saved job description'), expect.any(Function), expect.any(AbortSignal)))
  expect(api.updateResume).not.toHaveBeenCalled()
  expect(await screen.findByRole('status')).toHaveTextContent('Ready to review')
})

it('requires a job description before tailoring', async () => {
  vi.mocked(api.getJobs).mockResolvedValue([job({ description: '' })])
  vi.mocked(api.getJobResume).mockResolvedValue(null)
  const user = userEvent.setup()
  render(JobTweakView)
  await screen.findByRole('heading', { name: 'Allen' })
  await user.click(screen.getByRole('button', { name: /Engineer/ }))
  expect(await screen.findByRole('button', { name: 'Tailor resume' })).toBeDisabled()
  expect(screen.getByText('Add the job description above to start tailoring.')).toBeInTheDocument()
})

it('aborts tailoring when switching jobs and ignores late activity', async () => {
  vi.mocked(api.getJobResume).mockResolvedValue(resume({ id: 2, jobId: 7 }))
  let finish: () => void = () => {}
  let signal: AbortSignal | undefined
  vi.mocked(api.streamChat).mockImplementation(async (_id, _message, _event, abort) => {
    signal = abort
    await new Promise<void>(resolve => { finish = resolve })
  })
  const user = userEvent.setup()
  render(JobTweakView)
  await screen.findByRole('heading', { name: 'Allen' })
  await user.click(screen.getByRole('button', { name: /Engineer/ }))
  await user.click(await screen.findByRole('button', { name: 'Tailor resume' }))
  await waitFor(() => expect(signal).toBeDefined())
  await user.click(screen.getByRole('button', { name: /Base resume/ }))
  expect(signal?.aborted).toBe(true)
  finish()
  expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
})

it('refreshes the document after the assistant directly updates a resume', async () => {
  const updated = resume()
  updated.content.basics.headline = 'Senior Backend Engineer'
  vi.mocked(api.getBaseResume).mockResolvedValueOnce(resume()).mockResolvedValue(updated)
  vi.mocked(api.streamChat).mockImplementation(async (_id, _message, event) => {
    event({ type: 'tool', name: 'update_resume' })
    event({ type: 'text', text: 'Updated your headline.' })
  })
  const user = userEvent.setup()
  render(JobTweakView)
  await screen.findByRole('heading', { name: 'Allen' })
  await user.click(screen.getByRole('button', { name: /Ask AI/ }))
  await user.type(screen.getByLabelText('Message'), 'Update my headline{Enter}')
  expect(await screen.findByText('Senior Backend Engineer')).toBeInTheDocument()
  expect(api.acceptProposal).not.toHaveBeenCalled()
})

it('opens a persistent assistant sheet, closes with Escape, and removes the walkthrough', async () => {
  vi.mocked(api.streamChat).mockImplementation(async (_id, _message, event) => {
    event({ type: 'text', text: 'Your summary is updated.' })
  })
  const user = userEvent.setup()
  render(JobTweakView)
  await screen.findByRole('heading', { name: 'Allen' })
  expect(screen.queryByText('How it works')).not.toBeInTheDocument()
  expect(screen.queryByRole('dialog', { name: 'Resume assistant' })).not.toBeInTheDocument()
  const trigger = screen.getByRole('button', { name: /Ask AI/ })
  await user.click(trigger)
  expect(await screen.findByRole('dialog', { name: 'Resume assistant' })).toHaveAttribute('aria-modal', 'false')
  await user.type(screen.getByLabelText('Message'), 'Improve the summary{Enter}')
  expect(await screen.findByText('Your summary is updated.')).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Close assistant' }))
  expect(screen.queryByRole('dialog', { name: 'Resume assistant' })).not.toBeInTheDocument()
  await user.click(trigger)
  expect(screen.getByText('Your summary is updated.')).toBeVisible()
  await user.keyboard('{Escape}')
  await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Resume assistant' })).not.toBeInTheDocument())
})
