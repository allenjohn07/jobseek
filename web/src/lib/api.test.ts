import { afterEach, describe, expect, it, vi } from 'vitest'
import { createJob, deleteJob, getJobResume, getJobs, streamChat, updateJob, type ChatEvent } from './api'
import type { JobInput } from './types'

const input: JobInput = {
  company: 'Acme',
  title: 'Dev',
  url: '',
  status: 'wishlist',
  appliedOn: '',
  notes: '',
  description: '',
}

function mockFetch(res: Partial<Response> & { json?: () => unknown; text?: () => unknown }) {
  const fn = vi.fn().mockResolvedValue({ ok: true, status: 200, ...res })
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => vi.unstubAllGlobals())

describe('api', () => {
  it('getJobs returns parsed JSON', async () => {
    mockFetch({ json: async () => [{ id: 1 }] })
    expect(await getJobs()).toEqual([{ id: 1 }])
  })

  it('throws the server message on non-2xx', async () => {
    mockFetch({ ok: false, status: 400, text: async () => 'company and title are required' })
    await expect(createJob(input)).rejects.toThrow('company and title are required')
  })

  it('createJob POSTs JSON', async () => {
    const fn = mockFetch({ status: 201, json: async () => ({ id: 1, ...input }) })
    await createJob(input)
    const [url, init] = fn.mock.calls[0]
    expect(url).toBe('/api/jobs')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual(input)
  })

  it('updateJob PUTs the full job to /api/jobs/:id', async () => {
    const fn = mockFetch({ json: async () => ({ id: 7 }) })
    await updateJob(7, input)
    expect(fn.mock.calls[0][0]).toBe('/api/jobs/7')
    expect(fn.mock.calls[0][1].method).toBe('PUT')
  })

  it('deleteJob handles 204 without parsing a body', async () => {
    const fn = mockFetch({ status: 204 })
    await expect(deleteJob(3)).resolves.toBeUndefined()
    expect(fn.mock.calls[0][1].method).toBe('DELETE')
  })
})

describe('getJobResume', () => {
  it('returns null when the job has no resume yet (404)', async () => {
    mockFetch({ ok: false, status: 404 })
    expect(await getJobResume(1)).toBeNull()
  })

  it('throws the server message on other failures', async () => {
    mockFetch({ ok: false, status: 500, text: async () => 'boom' })
    await expect(getJobResume(1)).rejects.toThrow('boom')
  })
})

describe('streamChat', () => {
  const bodyOf = (chunks: string[]) =>
    new ReadableStream<Uint8Array>({
      start(c) {
        for (const ch of chunks) c.enqueue(new TextEncoder().encode(ch))
        c.close()
      },
    })

  it('parses events even when a chunk ends mid-event', async () => {
    const e1 = JSON.stringify({ type: 'text', text: 'Hi' })
    const e2 = JSON.stringify({ type: 'done' })
    const raw = `data: ${e1}\n\ndata: ${e2}\n\n`
    const cut = raw.indexOf('"Hi"') + 2 // split inside the first event
    mockFetch({ body: bodyOf([raw.slice(0, cut), raw.slice(cut)]) as unknown as Response['body'] })
    const got: ChatEvent[] = []
    await streamChat(1, 'hello', (e) => got.push(e))
    expect(got).toEqual([{ type: 'text', text: 'Hi' }, { type: 'done' }])
  })

  it('sends the resume and message', async () => {
    const fn = mockFetch({ body: bodyOf([]) as unknown as Response['body'] })
    await streamChat(5, 'tailor it', () => {})
    const [url, init] = fn.mock.calls[0]
    expect(url).toBe('/api/jobtweak/chat')
    expect(JSON.parse(init.body)).toEqual({ resumeId: 5, message: 'tailor it' })
  })

  it('throws the server message when the request is rejected', async () => {
    mockFetch({ ok: false, status: 409, text: async () => 'the AI is already working on this resume' })
    await expect(streamChat(1, 'x', () => {})).rejects.toThrow('already working')
  })
})
