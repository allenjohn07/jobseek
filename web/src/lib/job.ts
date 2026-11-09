import type { Job, JobInput } from './types'

// PUT /api/jobs/{id} replaces the whole job, so every update must send every field.
export const toInput = (j: Job): JobInput => ({
  company: j.company,
  title: j.title,
  url: j.url,
  status: j.status,
  appliedOn: j.appliedOn,
  notes: j.notes,
  description: j.description,
})
