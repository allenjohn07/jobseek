import type { JSONContent } from '@tiptap/core'
import type { ResumeContent } from './types'

const paragraph = (text: string, key?: string): JSONContent => ({ type: 'paragraph', attrs: key ? { resumeKey: key } : undefined, content: text ? [{ type: 'text', text }] : [] })
const heading = (text: string, level = 2, key?: string): JSONContent => ({ ...paragraph(text, key), type: 'heading', attrs: { level, ...(key ? { resumeKey: key } : {}) } })
const bullets = (items: { id: string; text: string }[]): JSONContent => ({ type: 'bulletList', content: items.map(item => ({ type: 'listItem', content: [paragraph(item.text, `item:${item.id}`)] })) })
export function toDocument(c: ResumeContent): JSONContent {
  if (c.document) return c.document
  const nodes: JSONContent[] = [heading(c.basics.name || 'Your name', 1, 'basics.name')]
  if (c.basics.headline) nodes.push(paragraph(c.basics.headline, 'basics.headline'))
  for (const key of ['email', 'phone', 'location', 'links'] as const) if (c.basics[key]) nodes.push(paragraph(c.basics[key], `basics.${key}`))
  if (c.summary.text) nodes.push(heading('Summary'), paragraph(c.summary.text, 'item:summary'))
  if (c.experience.length) {
    nodes.push(heading('Experience'))
    for (const e of c.experience) nodes.push({ type: 'resumeEntry', attrs: { entryId: e.id }, content: [heading(e.role, 3, `experience:${e.id}:role`), paragraph(e.company, `experience:${e.id}:company`), paragraph(e.start, `experience:${e.id}:start`), paragraph(e.end, `experience:${e.id}:end`), ...(e.bullets.length ? [bullets(e.bullets)] : [])] })
  }
  if (c.projects.length) {
    nodes.push(heading('Projects'))
    for (const e of c.projects) nodes.push({ type: 'resumeEntry', attrs: { entryId: e.id }, content: [heading(e.name, 3, `projects:${e.id}:name`), paragraph(e.link, `projects:${e.id}:link`), ...(e.bullets.length ? [bullets(e.bullets)] : [])] })
  }
  if (c.education.length) {
    nodes.push(heading('Education'))
    for (const e of c.education) nodes.push({ type: 'resumeEntry', attrs: { entryId: e.id }, content: [heading(e.school, 3, `education:${e.id}:school`), ...(['degree', 'start', 'end'] as const).map(key => paragraph(e[key], `education:${e.id}:${key}`))] })
  }
  if (c.skills.length) nodes.push(heading('Skills'), ...c.skills.map(item => paragraph(item.text, `item:${item.id}`)))
  return { type: 'doc', content: nodes }
}

export function documentText(node: JSONContent): string {
  if (node.type === 'text') return node.text ?? ''
  if (node.type === 'hardBreak') return '\n'
  return (node.content ?? []).map(documentText).join(['doc', 'resumeEntry', 'bulletList', 'orderedList', 'listItem', 'table', 'tableRow', 'tableCell', 'tableHeader', 'blockquote'].includes(node.type ?? '') ? '\n' : '')
}
