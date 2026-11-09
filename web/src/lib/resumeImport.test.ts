/// <reference types="node" />
import { readFile } from 'node:fs/promises'
import { describe, expect, it, vi } from 'vitest'
import { extractResume } from './resumeImport'

// Vitest loads Mammoth's Node entrypoint; adapt its input while exercising the real DOCX parser.
vi.mock('mammoth', async () => {
  const actual = await vi.importActual<typeof import('mammoth')>('mammoth')
  return { extractRawText: ({ arrayBuffer }: { arrayBuffer: ArrayBuffer }) => actual.extractRawText({ buffer: Buffer.from(arrayBuffer) }) }
})

describe('resume extraction', () => {
  it('extracts real DOCX paragraphs without converting to HTML', async () => {
    const data = await readFile('src/lib/test-fixtures/resume.docx')
    const file = { name: 'resume.docx', size: data.length, arrayBuffer: async () => data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) } as File
    expect(await extractResume(file)).toBe('Allen Example\n\nBackend developer. Built Go APIs.')
  })
  it('rejects oversized and unsupported files before parsing', async () => {
    await expect(extractResume({ name: 'resume.pdf', size: 11 * 1024 * 1024 } as File)).rejects.toThrow('10 MB')
    await expect(extractResume({ name: 'resume.doc', size: 12 } as File)).rejects.toThrow('PDF or DOCX')
  })
  it('explains corrupt documents and preserves a paste fallback', async () => {
    await expect(extractResume({ name: 'resume.docx', size: 12, arrayBuffer: async () => new ArrayBuffer(12) } as File)).rejects.toThrow('paste your resume text')
  })
})
