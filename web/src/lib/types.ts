// A "union type": status can ONLY be one of these exact strings.
// If you write status === 'aplied' (typo), TypeScript flags it as an error.
export type Status = 'wishlist' | 'applied' | 'interviewing' | 'offer' | 'rejected'

// An "interface" describes the shape of an object: which fields it has, and their types.
// These names must match the json tags in your Go Job struct exactly.
export interface Job {
  id: number         // Go int64 → number
  company: string
  title: string
  url: string
  status: Status     // uses the union type above instead of plain string
  appliedOn: string  // dates stay strings, e.g. "2026-11-05"
  notes: string
  createdAt: string
}

// Runtime list of all statuses (the type above only exists at compile time).
export const STATUSES: Status[] = ['wishlist', 'applied', 'interviewing', 'offer', 'rejected']

// What the form edits: a Job without the server-assigned fields.
export type JobInput = Omit<Job, 'id' | 'createdAt'>
