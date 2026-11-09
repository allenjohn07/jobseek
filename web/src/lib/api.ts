import type { Job, JobInput, McpInfo, Proposal, Resume, ResumeContent } from './types'

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

export const getBaseResume = () => send<Resume>('/api/resume/base', 'GET')
// null means this job has no tailored resume yet.
export async function getJobResume(jobId: number): Promise<Resume | null> {
  const res = await fetch(`/api/jobs/${jobId}/resume`)
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}
// Copies the base resume for this job.
export const createJobResume = (jobId: number) => send<Resume>(`/api/jobs/${jobId}/resume`, 'POST')
// Full replace; the server returns the content with IDs filled in.
export const updateResume = (id: number, content: ResumeContent) => send<Resume>(`/api/resumes/${id}`, 'PUT', content)

export const getProposals = (resumeId: number) => send<Proposal[]>(`/api/resumes/${resumeId}/proposals`, 'GET')
// Accepting applies the change to the resume; the server answers 409 if the item changed since.
export const acceptProposal = (id: number) => send<Proposal>(`/api/proposals/${id}/accept`, 'POST')
export const rejectProposal = (id: number) => send<Proposal>(`/api/proposals/${id}/reject`, 'POST')
export const getMcpInfo = () => send<McpInfo>('/api/mcp/info', 'GET')

// --- AI chat (server-sent events over a POST, so EventSource can't be used) ---

export type ChatEvent =
  | { type: 'notice'; message: string }
  | { type: 'text'; text: string }
  | { type: 'tool'; name: string }
  | { type: 'done' }
  | { type: 'error'; message: string }

// Sends a message and calls onEvent for each streamed event. Aborting the signal stops the AI.
export async function streamChat(
  resumeId: number,
  message: string,
  onEvent: (e: ChatEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch('/api/jobtweak/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ resumeId, message }),
    signal,
  })
  if (!res.ok) throw new Error(await res.text())
  if (!res.body) throw new Error('The server sent no response stream')

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    // Events end with a blank line; a chunk can end mid-event, so keep the remainder.
    let end: number
    while ((end = buffer.indexOf('\n\n')) >= 0) {
      const frame = buffer.slice(0, end)
      buffer = buffer.slice(end + 2)
      for (const line of frame.split('\n')) {
        if (line.startsWith('data: ')) onEvent(JSON.parse(line.slice(6)))
      }
    }
  }
}

export async function importResume(text: string, signal?: AbortSignal): Promise<ResumeContent> {
  const res = await fetch('/api/resume/import', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }), signal,
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}
