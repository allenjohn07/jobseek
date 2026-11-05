import { afterEach, describe, expect, it, vi } from 'vitest'
import { createJob, deleteJob, getJobs, updateJob } from './api'
import type { JobInput } from './types'

const input: JobInput = {
  company: 'Acme',
  title: 'Dev',
  url: '',
  status: 'wishlist',
  appliedOn: '',
  notes: '',
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
