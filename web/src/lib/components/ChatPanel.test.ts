import { render, screen, waitFor } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import ChatPanel from './ChatPanel.svelte'
import * as api from '$lib/api'

vi.mock('$lib/api')

const setup = (props: { jobTitle?: string } = {}) => {
  const onActivity = vi.fn()
  render(ChatPanel, { resumeId: 4, onActivity, ...props })
  return { onActivity, user: userEvent.setup() }
}

describe('ChatPanel', () => {
  it('sends the message, streams the reply, and shows tool activity', async () => {
    vi.mocked(api.streamChat).mockImplementation(async (_id, _msg, onEvent) => {
      onEvent({ type: 'text', text: 'Looking at it' })
      onEvent({ type: 'text', text: ' now.' })
      onEvent({ type: 'tool', name: 'propose_edit' })
      onEvent({ type: 'text', text: 'Suggested one change.' })
      onEvent({ type: 'done' })
    })
    const { user, onActivity } = setup()

    await user.type(screen.getByLabelText('Message'), 'make it shorter')
    await user.click(screen.getByRole('button', { name: 'Send' }))

    expect(api.streamChat).toHaveBeenCalledWith(4, 'make it shorter', expect.any(Function), expect.any(AbortSignal))
    expect(screen.getByText('make it shorter')).toBeInTheDocument()
    expect(await screen.findByText('Looking at it now.')).toBeInTheDocument() // deltas join into one paragraph
    expect(screen.getByText(/Suggested an edit/)).toBeInTheDocument()
    expect(screen.getByText('Suggested one change.')).toBeInTheDocument()
    // Refreshes suggestions when one arrives and again when the turn ends.
    await waitFor(() => expect(onActivity).toHaveBeenCalledTimes(2))
    expect(screen.getByLabelText('Message')).toHaveValue('')
  })

  it('cannot send an empty message', async () => {
    setup()
    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
  })

  it('Enter sends and Shift+Enter adds a new line', async () => {
    vi.mocked(api.streamChat).mockResolvedValue()
    const { user } = setup()
    const box = screen.getByLabelText('Message')

    await user.type(box, 'line one{Shift>}{Enter}{/Shift}line two')
    expect(api.streamChat).not.toHaveBeenCalled()
    expect(box).toHaveValue('line one\nline two')

    await user.keyboard('{Enter}')
    await waitFor(() => expect(api.streamChat).toHaveBeenCalledWith(4, 'line one\nline two', expect.any(Function), expect.any(AbortSignal)))
  })

  it('offers a one-click tailoring request when a job is selected', async () => {
    vi.mocked(api.streamChat).mockResolvedValue()
    const { user } = setup({ jobTitle: 'Go Engineer' })
    await user.click(screen.getByRole('button', { name: /Tailor my resume for this job/ }))
    expect(api.streamChat).toHaveBeenCalledWith(4, 'Tailor my resume for the Go Engineer job.', expect.any(Function), expect.any(AbortSignal))
  })

  it('shows an error event from the AI', async () => {
    vi.mocked(api.streamChat).mockImplementation(async (_i, _m, onEvent) => {
      onEvent({ type: 'error', message: "Claude Code isn't installed." })
    })
    const { user } = setup()
    await user.type(screen.getByLabelText('Message'), 'hi{Enter}')
    expect(await screen.findByRole('alert')).toHaveTextContent("Claude Code isn't installed.")
  })

  it('shows the server message when the request is rejected', async () => {
    vi.mocked(api.streamChat).mockRejectedValue(new Error('the AI is already working on this resume'))
    const { user } = setup()
    await user.type(screen.getByLabelText('Message'), 'hi{Enter}')
    expect(await screen.findByRole('alert')).toHaveTextContent('already working')
    expect(screen.getByRole('button', { name: 'Send' })).toBeInTheDocument() // can retry
  })

  it('Stop aborts the request without showing an error', async () => {
    vi.mocked(api.streamChat).mockImplementation(
      (_i, _m, _e, signal) =>
        new Promise((_res, rej) => signal?.addEventListener('abort', () => rej(new DOMException('aborted', 'AbortError')))),
    )
    const { user, onActivity } = setup()
    await user.type(screen.getByLabelText('Message'), 'hi{Enter}')

    await user.click(await screen.findByRole('button', { name: 'Stop' }))
    expect(await screen.findByRole('button', { name: 'Send' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(onActivity).toHaveBeenCalled()
  })

  it('ignores a second message while the AI is working', async () => {
    let finish: () => void = () => {}
    vi.mocked(api.streamChat).mockImplementation(() => new Promise<void>((res) => (finish = res)))
    const { user } = setup()
    await user.type(screen.getByLabelText('Message'), 'first{Enter}')
    expect(screen.getByLabelText('Message')).toBeDisabled()
    expect(api.streamChat).toHaveBeenCalledTimes(1)
    finish()
    await waitFor(() => expect(screen.getByLabelText('Message')).not.toBeDisabled())
  })
})

it('positions each new reply once and does not scroll on every streamed chunk', async () => {
  const scroll = vi.fn()
  const original = HTMLElement.prototype.scrollTo
  HTMLElement.prototype.scrollTo = scroll
  let emit: (event: api.ChatEvent) => void = () => {}
  let finish = () => {}
  vi.mocked(api.streamChat).mockImplementation((_id, _msg, event) => {
    emit = event
    return new Promise<void>(resolve => { finish = resolve })
  })
  try {
    const { user } = setup()
    await user.type(screen.getByLabelText('Message'), 'Improve my summary{Enter}')
    await waitFor(() => expect(scroll).toHaveBeenCalledTimes(1))
    emit({ type: 'text', text: 'Updated the summary.' })
    emit({ type: 'text', text: ' Your experience is unchanged.' })
    await screen.findByText('Updated the summary. Your experience is unchanged.')
    expect(scroll).toHaveBeenCalledTimes(1)
    finish()
    await waitFor(() => expect(screen.getByLabelText('Message')).not.toBeDisabled())
    await user.type(screen.getByLabelText('Message'), 'Shorten it further{Enter}')
    await waitFor(() => expect(scroll).toHaveBeenCalledTimes(2))
    finish()
  } finally { HTMLElement.prototype.scrollTo = original }
})

it('explains when an old AI conversation is refreshed for updated editing tools', async () => {
  vi.mocked(api.streamChat).mockImplementation(async (_id, _msg, event) => {
    event({ type: 'notice', message: 'Updated resume-editing tools loaded in a fresh conversation.' })
    event({ type: 'text', text: 'I can reorder your sections.' })
  })
  const { user } = setup()
  await user.type(screen.getByLabelText('Message'), 'Reorder the sections{Enter}')
  expect(await screen.findByRole('status')).toHaveTextContent('Updated resume-editing tools loaded')
  expect(screen.getByText('I can reorder your sections.')).toBeInTheDocument()
})
