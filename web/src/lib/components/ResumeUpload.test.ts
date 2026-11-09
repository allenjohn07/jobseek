import { render, screen, waitFor } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import ResumeUpload from './ResumeUpload.svelte'
import { importResume } from '$lib/api'
import { extractResume } from '$lib/resumeImport'
import type { ResumeContent } from '$lib/types'
vi.mock('$lib/api')
vi.mock('$lib/resumeImport')
const content: ResumeContent = { basics: { name: 'Allen', headline: '', email: '', phone: '', location: '', links: '' }, summary: { id: 'summary', text: 'Developer' }, experience: [], education: [], skills: [], projects: [] }
beforeEach(() => { vi.mocked(importResume).mockResolvedValue(structuredClone(content)) })
it('reviews imported content and only saves after confirmation', async () => {
  const save = vi.fn().mockResolvedValue(undefined)
  const user = userEvent.setup()
  render(ResumeUpload, { onSave: save, onCancel: vi.fn() })
  await user.type(screen.getByLabelText('Or paste your resume text'), 'Allen Developer')
  await user.click(screen.getByRole('button', { name: 'Organize with AI' }))
  expect(await screen.findByDisplayValue('Allen')).toBeInTheDocument()
  expect(save).not.toHaveBeenCalled()
  expect(screen.getByLabelText('Original resume text')).toHaveValue('Allen Developer')
  await user.clear(screen.getByLabelText('Name'))
  await user.type(screen.getByLabelText('Name'), 'Allen John{Enter}')
  await user.click(screen.getByRole('button', { name: 'Save base resume' }))
  expect(save).toHaveBeenCalledWith(expect.objectContaining({ basics: expect.objectContaining({ name: 'Allen John' }) }))
})
it('extracts a selected document and retains errors without saving', async () => {
  vi.mocked(extractResume).mockResolvedValue('Allen Developer')
  vi.mocked(importResume).mockRejectedValue(new Error('Claude sign-in required'))
  const save = vi.fn()
  const user = userEvent.setup()
  render(ResumeUpload, { onSave: save, onCancel: vi.fn() })
  await user.upload(screen.getByLabelText('Upload resume'), new File(['pdf'], 'resume.pdf', { type: 'application/pdf' }))
  await waitFor(() => expect(screen.getByLabelText('Or paste your resume text')).toHaveValue('Allen Developer'))
  await user.click(screen.getByRole('button', { name: 'Organize with AI' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Claude sign-in required')
  expect(save).not.toHaveBeenCalled()
})
it('cancels an in-flight import', async () => {
  vi.mocked(importResume).mockImplementation(() => new Promise(() => {}))
  const cancel = vi.fn()
  const user = userEvent.setup()
  render(ResumeUpload, { onSave: vi.fn(), onCancel: cancel })
  await user.type(screen.getByLabelText('Or paste your resume text'), 'Allen')
  await user.click(screen.getByRole('button', { name: 'Organize with AI' }))
  await user.click(screen.getByRole('button', { name: 'Cancel' }))
  expect(cancel).toHaveBeenCalledOnce()
  expect(vi.mocked(importResume).mock.lastCall?.[1]?.aborted).toBe(true)
})
