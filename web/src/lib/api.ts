import type { Job } from './types'  // "import type" imports only the type; it disappears from the built JS

// ": Promise<Job[]>" is the return type: this async function eventually gives back an array of Jobs.
// Now anywhere you call getJobs(), your editor knows exactly what's inside.
export async function getJobs(): Promise<Job[]> {
  const res = await fetch('/api/jobs')           // GET /api/jobs
  if (!res.ok) throw new Error(await res.text()) // non-2xx status → throw with the server's message
  return res.json()                              // parse the JSON into Job[]
}