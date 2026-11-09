import { act, render, screen, waitFor } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Editor } from '@tiptap/core'
import ResumeCanvas from './ResumeCanvas.svelte'
import type { ResumeContent } from '$lib/types'
const base = (): ResumeContent => ({ basics: { name: 'Allen', headline: 'Developer', email: 'allen@example.com', phone: '', location: '', links: '' }, summary: { id: 'summary', text: 'Build reliable services' }, experience: [], education: [], skills: [], projects: [] })
const editor = () => (screen.getByRole('textbox', { name: 'Resume document' }) as HTMLElement & { editor: Editor }).editor

describe('document editor', () => {
  it('opens existing resumes as an editable page without changing saved data', async () => {
    const save = vi.fn()
    render(ResumeCanvas, { content: base(), onChange: save })
    expect(await screen.findByRole('heading', { name: 'Allen' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Resume document' })).toHaveAttribute('contenteditable', 'true')
    expect(screen.getByRole('toolbar', { name: 'Document formatting' })).toBeInTheDocument()
    expect(save).not.toHaveBeenCalled()
  })
  it('persists rich text and tables, then renders them in read-only preview', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const view = render(ResumeCanvas, { content: base(), onChange: save })
    await screen.findByRole('textbox', { name: 'Resume document' })
    await act(() => {
      editor().chain().setTextSelection({ from: 1, to: 6 }).toggleBold().run()
      editor().commands.insertContentAt(editor().state.doc.content.size, { type: 'table', content: [{ type: 'tableRow', content: [{ type: 'tableCell', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'New achievement' }] }] }] }] })
    })
    await waitFor(() => expect(save).toHaveBeenCalled())
    const content = save.mock.lastCall![0] as ResumeContent
    expect(JSON.stringify(content.document)).toContain('"bold"')
    expect(JSON.stringify(content.document)).toContain('New achievement')
    view.unmount()
    render(ResumeCanvas, { content, editable: false })
    expect(await screen.findByRole('cell', { name: 'New achievement' })).toBeInTheDocument()
    expect(screen.queryByRole('toolbar')).not.toBeInTheDocument()
    expect(screen.getByRole('document', { name: 'Resume preview' })).toHaveAttribute('contenteditable', 'false')
  })
  it('flushes unsaved edits before the caller continues', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { component } = render(ResumeCanvas, { content: base(), onChange: save })
    await screen.findByRole('textbox', { name: 'Resume document' })
    await act(() => { editor().commands.insertContentAt(1, 'Edited ') })
    expect(save).not.toHaveBeenCalled()
    await component.flush()
    expect(save).toHaveBeenCalledOnce()
    expect(JSON.stringify(save.mock.lastCall![0].document)).toContain('Edited Allen')
  })
  it('allows formatting using the toolbar without removing content', async () => {
    render(ResumeCanvas, { content: base(), onChange: vi.fn().mockResolvedValue(undefined) })
    await screen.findByRole('textbox', { name: 'Resume document' })
    await act(() => { editor().commands.setTextSelection({ from: 1, to: 6 }) })
    await userEvent.click(screen.getByRole('button', { name: 'Bold' }))
    expect(document.querySelector('.resume-prose h1 strong')).toHaveTextContent('Allen')
  })
})
