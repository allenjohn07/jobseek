import { Extension, Node, mergeAttributes } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { newId } from './resume'
import StarterKit from '@tiptap/starter-kit'
import { TableKit } from '@tiptap/extension-table'
import TextAlign from '@tiptap/extension-text-align'

const ResumeKeys = Extension.create({
  name: 'resumeKeys',
  addProseMirrorPlugins() {
    return [new Plugin({
      key: new PluginKey('stableResumeKeys'),
      appendTransaction(transactions, _oldState, state) {
        if (!transactions.some(transaction => transaction.docChanged)) return null
        const transaction = state.tr
        const seen = new Set<string>()
        state.doc.descendants((node, pos) => {
          if (!['paragraph', 'heading'].includes(node.type.name)) return
          let key = node.attrs.resumeKey as string | null
          if (!key || seen.has(key)) {
            key = `item:document-${newId()}`
            transaction.setNodeMarkup(pos, undefined, { ...node.attrs, resumeKey: key })
          }
          seen.add(key)
        })
        return transaction.docChanged ? transaction : null
      },
    })]
  },
  addGlobalAttributes() {
    return [{ types: ['paragraph', 'heading'], attributes: { resumeKey: {
      default: null,
      parseHTML: element => element.getAttribute('data-resume-key'),
      renderHTML: attrs => attrs.resumeKey ? { 'data-resume-key': attrs.resumeKey } : {},
    } } }]
  },
})
const ResumeEntry = Node.create({
  name: 'resumeEntry', group: 'block', content: 'block+', defining: true,
  addAttributes() { return { entryId: { default: null, parseHTML: element => element.getAttribute('data-entry-id'), renderHTML: attrs => ({ 'data-entry-id': attrs.entryId }) } } },
  parseHTML() { return [{ tag: 'section[data-entry-id]' }] },
  renderHTML({ HTMLAttributes }) { return ['section', mergeAttributes(HTMLAttributes), 0] },
})
export const documentExtensions = () => [StarterKit.configure({ link: { openOnClick: false }, heading: { levels: [1, 2, 3] } }), TableKit.configure({ table: { resizable: false } }), TextAlign.configure({ types: ['heading', 'paragraph'] }), ResumeKeys, ResumeEntry]
