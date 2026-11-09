import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import ProposalCard from './ProposalCard.svelte'
import type { Proposal, ResumeContent } from '$lib/types'
const snapshot = (order: string[]): ResumeContent => ({ basics: { name: 'Allen', headline: '', email: '', phone: '', location: '', links: '' }, summary: { id: 'summary', text: '' }, experience: [], education: [], projects: [], skills: [], document: { type: 'doc', content: order.flatMap(section => [{ type: 'heading', attrs: { level: 2 }, content: [{ type: 'text', text: section }] }, { type: 'paragraph', content: [{ type: 'text', text: `${section} details` }] }]) } })
it('reviews a section reorder with full before/after previews and acceptance controls', async () => {
  const proposal: Proposal = { id: 1, resumeId: 1, op: 'replace_document', itemId: '', beforeText: JSON.stringify(snapshot(['Experience', 'Education'])), afterText: JSON.stringify(snapshot(['Education', 'Experience'])), reason: 'Lead with education', status: 'pending', createdAt: '' }
  const accept = vi.fn(); const reject = vi.fn(); const user = userEvent.setup()
  render(ProposalCard, { proposal, onAccept: accept, onReject: reject })
  expect(screen.getByText(/Experience → Education/)).toBeInTheDocument()
  expect(screen.getByText(/Education → Experience/)).toBeInTheDocument()
  await user.click(screen.getByText('Review full document'))
  expect(await screen.findByText('Current resume')).toBeInTheDocument()
  expect(screen.getByText('Proposed resume')).toBeInTheDocument()
  expect(screen.getAllByText('Education details')).toHaveLength(2)
  await user.click(screen.getByRole('button', { name: 'Accept' }))
  expect(accept).toHaveBeenCalledOnce()
  await user.click(screen.getByRole('button', { name: 'Reject' }))
  expect(reject).toHaveBeenCalledOnce()
})
