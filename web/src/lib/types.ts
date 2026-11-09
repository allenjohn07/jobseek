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
  description: string // the job posting text, used to tailor the resume
  createdAt: string
}

// Runtime list of all statuses (the type above only exists at compile time).
export const STATUSES: Status[] = ['wishlist', 'applied', 'interviewing', 'offer', 'rejected']

// What the form edits: a Job without the server-assigned fields.
export type JobInput = Omit<Job, 'id' | 'createdAt'>

// --- Resume (mirrors resume.go) ---

// One editable piece of text. The id is stable so AI suggestions can target it.
export interface Item {
  id: string
  text: string
}
export interface Basics {
  name: string
  headline: string
  email: string
  phone: string
  location: string
  links: string
}
export interface Experience {
  id: string
  company: string
  role: string
  start: string
  end: string
  bullets: Item[]
}
export interface Education {
  id: string
  school: string
  degree: string
  start: string
  end: string
}
export interface Project {
  id: string
  name: string
  link: string
  bullets: Item[]
}
export interface ResumeContent {
  document?: import('@tiptap/core').JSONContent
  basics: Basics
  summary: Item
  experience: Experience[]
  education: Education[]
  skills: Item[]
  projects: Project[]
}
// jobId is null for the base resume.
export interface Resume {
  revision?: string // used by AI tools for conflict-safe updates
  id: number
  jobId: number | null
  content: ResumeContent
  sessionId: string
  updatedAt: string
}

// A pending AI suggestion. Only the user can accept it (never the AI).
export interface Proposal {
  id: number
  resumeId: number
  op: 'replace' | 'insert_after' | 'delete' | 'replace_document'
  itemId: string
  beforeText: string
  afterText: string
  reason: string
  status: 'pending' | 'accepted' | 'rejected'
  createdAt: string
}

// What the Connect dialog shows so you can add this app to Claude Code.
export interface McpInfo {
  url: string
  token: string
  command: string
}
