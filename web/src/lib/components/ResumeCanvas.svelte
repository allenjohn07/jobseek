<script lang="ts">
  import { onMount, onDestroy, untrack } from 'svelte'
  import { Editor, type JSONContent } from '@tiptap/core'
  import { documentExtensions } from '$lib/documentExtensions'
  import { toDocument, documentText } from '$lib/resumeDocument'
  import type { ResumeContent } from '$lib/types'
  import { Button } from './ui/button'
  import Bold from '@lucide/svelte/icons/bold'
  import Italic from '@lucide/svelte/icons/italic'
  import Underline from '@lucide/svelte/icons/underline'
  import List from '@lucide/svelte/icons/list'
  import ListOrdered from '@lucide/svelte/icons/list-ordered'
  import Undo from '@lucide/svelte/icons/undo-2'
  import Redo from '@lucide/svelte/icons/redo-2'
  import Table from '@lucide/svelte/icons/table'
  import AlignLeft from '@lucide/svelte/icons/align-left'
  import AlignCenter from '@lucide/svelte/icons/align-center'
  import AlignRight from '@lucide/svelte/icons/align-right'
  let { content, onChange, editable = true, printable = true }: { content: ResumeContent; onChange?: (next: ResumeContent) => Promise<void> | void; editable?: boolean; printable?: boolean } = $props()
  let host = $state<HTMLDivElement>()
  let editor = $state.raw<Editor>()
  let transaction = $state(0)
  let dirty = $state(false)
  let status = $state('All changes saved')
  let failure = $state('')
  let timer: ReturnType<typeof setTimeout> | undefined
  let saving: Promise<void> | null = null
  let acknowledged = ''
  let lastSubmitted = ''
  const active = (name: string) => { void transaction; return editor?.isActive(name) ?? false }
  onMount(() => {
    acknowledged = JSON.stringify(toDocument(content))
    editor = new Editor({ element: host, extensions: documentExtensions(), content: toDocument(content), editable,
      editorProps: { handleKeyDown: (_view, event) => { if ((event.metaKey || event.ctrlKey) && event.key === 's') { event.preventDefault(); void flush().catch(() => {}); return true } return false }, attributes: { class: 'resume-prose', role: editable ? 'textbox' : 'document', 'aria-label': editable ? 'Resume document' : 'Resume preview', 'aria-multiline': 'true' } },
      onTransaction: () => { transaction++ },
      onUpdate: () => { if (!editable) return; dirty = true; status = 'Unsaved changes'; clearTimeout(timer); timer = setTimeout(() => { void flush().catch(() => {}) }, 650) },
      onBlur: () => { if (dirty) void flush().catch(() => {}) },
    })
  })
  onDestroy(() => { clearTimeout(timer); if (dirty) void flush().catch(() => {}); editor?.destroy() })
  $effect(() => {
    const next = JSON.stringify(toDocument(content))
    const current = editor
    if (!current || next === acknowledged || next === lastSubmitted) return
    // Server echoes never reset selection or undo history. External AI changes arrive after flush().
    if (untrack(() => dirty || saving !== null)) return
    acknowledged = next
    if (JSON.stringify(current.getJSON()) !== next) current.commands.setContent(JSON.parse(next), { emitUpdate: false })
  })
  export async function flush(): Promise<void> {
    clearTimeout(timer)
    if (saving) { await saving; if (dirty) return flush(); return }
    if (!editor || !dirty || !onChange) return
    const document = editor.getJSON()
    lastSubmitted = JSON.stringify(document)
    dirty = false; status = 'Saving…'; failure = ''
    const next = { ...JSON.parse(JSON.stringify(content)), document } as ResumeContent
    saving = (async () => {
      try { await onChange(next); acknowledged = lastSubmitted; status = dirty ? 'Unsaved changes' : 'All changes saved' }
      catch (e) { dirty = true; failure = e instanceof Error ? e.message : String(e); status = 'Could not save'; throw e }
      finally { saving = null }
    })()
    await saving
    if (dirty) await flush()
  }
  function insertTable() { editor?.chain().focus().insertTable({ rows: 3, cols: 2, withHeaderRow: true }).run() }
  function block(value: string) {
    if (!editor) return
    if (value === 'paragraph') editor.chain().focus().setParagraph().run()
    else editor.chain().focus().toggleHeading({ level: Number(value) as 1 | 2 | 3 }).run()
  }
</script>

<div class="resume-canvas {editable ? 'bg-muted/25' : printable ? 'resume-print' : ''}">
  {#if editable}
    <div role="toolbar" tabindex="-1" aria-label="Document formatting" class="sticky top-0 z-10 flex flex-wrap items-center gap-1 border-b bg-background/95 px-3 py-2 backdrop-blur" onmousedown={(event) => { if (!(event.target instanceof HTMLSelectElement)) event.preventDefault() }}>
      <select aria-label="Text style" class="mr-1 rounded-md border bg-background px-2 py-1.5 text-xs" onchange={(event) => block(event.currentTarget.value)} value={transaction >= 0 && editor?.isActive('heading') ? String(editor?.getAttributes('heading').level) : 'paragraph'}><option value="paragraph">Normal text</option><option value="1">Title</option><option value="2">Heading</option><option value="3">Subheading</option></select>
      {#each [{ label: 'Bold', icon: Bold, name: 'bold', run: () => editor?.chain().focus().toggleBold().run() }, { label: 'Italic', icon: Italic, name: 'italic', run: () => editor?.chain().focus().toggleItalic().run() }, { label: 'Underline', icon: Underline, name: 'underline', run: () => editor?.chain().focus().toggleUnderline().run() }, { label: 'Bullet list', icon: List, name: 'bulletList', run: () => editor?.chain().focus().toggleBulletList().run() }, { label: 'Numbered list', icon: ListOrdered, name: 'orderedList', run: () => editor?.chain().focus().toggleOrderedList().run() }] as item}
        <Button variant="ghost" size="icon-sm" title={item.label} aria-label={item.label} aria-pressed={active(item.name)} class={active(item.name) ? 'bg-accent' : ''} onclick={item.run}><item.icon class="size-4" /></Button>
      {/each}
      <span class="mx-1 h-5 border-l"></span>
      {#each [{ value: 'left', label: 'Align left', icon: AlignLeft }, { value: 'center', label: 'Align center', icon: AlignCenter }, { value: 'right', label: 'Align right', icon: AlignRight }] as alignment}
        <Button variant="ghost" size="icon-sm" aria-label={alignment.label} title={alignment.label} aria-pressed={transaction >= 0 && (editor?.isActive({ textAlign: alignment.value }) ?? false)} onclick={() => editor?.chain().focus().setTextAlign(alignment.value).run()}><alignment.icon class="size-4" /></Button>
      {/each}
      <span class="mx-1 h-5 border-l"></span>
      <Button variant="ghost" size="icon-sm" title="Insert table" aria-label="Insert table" onclick={insertTable}><Table class="size-4" /></Button>
      {#if active('table')}<Button variant="ghost" class="text-xs" onclick={() => editor?.chain().focus().addRowAfter().run()}>Add row</Button><Button variant="ghost" class="text-xs" onclick={() => editor?.chain().focus().addColumnAfter().run()}>Add column</Button><Button variant="ghost" class="text-xs" onclick={() => editor?.chain().focus().deleteTable().run()}>Remove table</Button>{/if}
      <Button variant="ghost" size="icon-sm" title="Undo" aria-label="Undo" onclick={() => editor?.chain().focus().undo().run()}><Undo class="size-4" /></Button>
      <Button variant="ghost" size="icon-sm" title="Redo" aria-label="Redo" onclick={() => editor?.chain().focus().redo().run()}><Redo class="size-4" /></Button>
      <span class="ml-auto px-2 text-[11px] text-muted-foreground" aria-live="polite">{status}</span>
      {#if failure}<Button variant="outline" class="text-xs" onclick={() => flush().catch(() => {})}>Retry save</Button>{/if}
    </div>
    {#if failure}<p role="alert" class="px-4 pt-3 text-sm text-destructive">{failure}. Your edits are still here; retry saving before leaving.</p>{/if}
  {/if}
  <div class={editable ? 'px-2 py-5 sm:px-5 sm:py-8' : ''}>
    <div bind:this={host} class="document-page mx-auto w-full max-w-[50rem] bg-background {editable ? 'min-h-[65rem] border px-5 py-8 shadow-sm sm:px-12 sm:py-12' : 'px-5 py-8 sm:px-10'}"></div>
  </div>
  {#if editable}<div class="flex justify-between border-t bg-background px-4 py-2 text-[11px] text-muted-foreground"><span>Click anywhere to edit · Paste text, add sections, or use the toolbar</span><span>{transaction >= 0 && editor ? documentText(editor.getJSON()).trim().split(/\s+/).filter(Boolean).length : 0} words</span></div>{/if}
</div>
