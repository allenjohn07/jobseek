import type { Job, JobInput } from './types'

// ": Promise<Job[]>" is the return type: this async function eventually gives back an array of Jobs.
// Now anywhere you call getJobs(), your editor knows exactly what's inside.
export async function getJobs(): Promise<Job[]> {
  const res = await fetch('/api/jobs')           // GET /api/jobs
  if (!res.ok) throw new Error(await res.text()) // non-2xx status → throw with the server's message
  return res.json()                              // parse the JSON into Job[]
}
// Shared helper: send JSON and throw the server's plain-text error on failure.
async function send<T>(url: string, method: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) throw new Error(await res.text())
  return (res.status === 204 ? undefined : await res.json()) as T
}

export const createJob = (job: JobInput) => send<Job>('/api/jobs', 'POST', job)
// PUT is a full replace on the Go side, so always send every field.
export const updateJob = (id: number, job: JobInput) => send<Job>(`/api/jobs/${id}`, 'PUT', job)
export const deleteJob = (id: number) => send<void>(`/api/jobs/${id}`, 'DELETE')
