import type { Education, Experience, Item, Project, ResumeContent } from './types'

// Client-side IDs for newly added items. The server keeps any non-empty ID it receives.
export const newId = () => Math.random().toString(36).slice(2, 10) + Date.now().toString(36)

export const newItem = (text = ''): Item => ({ id: newId(), text })
export const newExperience = (): Experience => ({ id: newId(), company: '', role: '', start: '', end: '', bullets: [] })
export const newEducation = (): Education => ({ id: newId(), school: '', degree: '', start: '', end: '' })
export const newProject = (): Project => ({ id: newId(), name: '', link: '', bullets: [] })

// Deep copy of plain data (also unwraps Svelte $state proxies' values).
export const clone = (c: ResumeContent): ResumeContent => JSON.parse(JSON.stringify(c))

export const dateRange = (start: string, end: string) => [start, end].filter(Boolean).join(' – ')

export const hasContent = (c: ResumeContent) => !!c.document || Object.values(c.basics).some(value => value.trim()) || !!c.summary.text.trim() || c.experience.length > 0 || c.education.length > 0 || c.skills.some(item => item.text.trim()) || c.projects.length > 0
