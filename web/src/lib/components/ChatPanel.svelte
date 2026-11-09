<script lang="ts">
  import { onDestroy, tick } from 'svelte'
  import { streamChat, type ChatEvent } from '$lib/api'
  import { Button } from '$lib/components/ui/button'
  import ErrorBanner from '$lib/components/ErrorBanner.svelte'
  import SparklesIcon from '@lucide/svelte/icons/sparkles'
  import XIcon from '@lucide/svelte/icons/x'

  // Refresh the document after agent tools complete and when the turn finishes.
  let {
    resumeId,
    jobTitle = '',
    onActivity,
    fillHeight = false,
    onClose,
    beforeSend,
  }: { resumeId: number; jobTitle?: string; onActivity: () => void; fillHeight?: boolean; onClose?: () => void; beforeSend?: () => Promise<void> } = $props()

  type Part = { kind: 'text'; text: string } | { kind: 'tool'; label: string }
  type Msg = { role: 'user'; text: string } | { role: 'assistant'; parts: Part[] }

  const toolLabels: Record<string, string> = {
    list_jobs: 'Looking at your jobs',
    get_job: 'Reading the job description',
    get_resume: 'Reading the resume',
    create_job_resume: 'Making a tailored copy',
    propose_edit: 'Suggested an edit',
    update_resume: 'Updating your resume',
    list_proposals: 'Checking earlier suggestions',
  }

  let messages = $state<Msg[]>([])
  let draft = $state('')
  let running = $state(false)
  let error = $state('')
  let notice = $state('')
  let controller: AbortController | null = null
  let log = $state<HTMLDivElement>()
  let logHeight = $state(400)
  onDestroy(() => controller?.abort())
  function measure(node: HTMLDivElement) {
    const observer = new ResizeObserver(() => { logHeight = node.clientHeight })
    observer.observe(node)
    logHeight = node.clientHeight || 400
    return { destroy: () => observer.disconnect() }
  }
  function anchorReply(node: HTMLDivElement) {
    let active = true
    tick().then(() => {
      if (!active || !log) return
      // Position the start of the reply once; streamed text grows below without moving the reader.
      const top = node.getBoundingClientRect().top - log.getBoundingClientRect().top + log.scrollTop - 20
      log.scrollTo?.({ top: Math.max(0, top), behavior: 'smooth' })
    })
    return { destroy: () => { active = false } }
  }

  function onEvent(ev: ChatEvent, reply: Extract<Msg, { role: 'assistant' }>) {
    if (ev.type === 'notice') {
      notice = ev.message
    } else if (ev.type === 'text') {
      const last = reply.parts.at(-1)
      if (last?.kind === 'text') last.text += ev.text
      else reply.parts.push({ kind: 'text', text: ev.text })
    } else if (ev.type === 'tool') {
      reply.parts.push({ kind: 'tool', label: toolLabels[ev.name] ?? ev.name })
      if (ev.name === 'propose_edit') onActivity() // show suggestions as they arrive
    } else if (ev.type === 'error') {
      error = ev.message
    }
  }

  async function send(text: string) {
    text = text.trim()
    if (!text || running) return
    error = ''
    if (beforeSend) {
      running = true
      try { await beforeSend() } catch (e) { error = e instanceof Error ? e.message : String(e); running = false; return }
      running = false
    }
    draft = ''
    messages.push({ role: 'user', text })
    messages.push({ role: 'assistant', parts: [] })
    // Take the proxied object back out of the array so updates re-render.
    const reply = messages[messages.length - 1] as Extract<Msg, { role: 'assistant' }>
    running = true
    controller = new AbortController()
    try {
      await streamChat(resumeId, text, (ev) => onEvent(ev, reply), controller.signal)
    } catch (e) {
      if (!(e instanceof DOMException && e.name === 'AbortError')) error = e instanceof Error ? e.message : String(e)
    } finally {
      running = false
      controller = null
      onActivity()
    }
  }

  const stop = () => controller?.abort()

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
      e.preventDefault()
      send(draft)
    }
  }
</script>

<section aria-label="AI chat" class="flex flex-col overflow-hidden bg-background {fillHeight ? 'h-full min-h-0 flex-1' : 'h-[clamp(28rem,calc(100dvh-25rem),48rem)] min-h-[28rem] rounded-xl border shadow-sm'}">
  <header class="flex shrink-0 items-center gap-3 border-b bg-muted/25 px-5 py-4">
    <span class="flex size-9 items-center justify-center rounded-xl border bg-background"><SparklesIcon class="size-4" /></span>
    <div class="min-w-0 flex-1"><h3 class="text-sm font-semibold">Resume assistant</h3><p class="mt-0.5 text-xs text-muted-foreground">Changes saved directly to this resume</p></div>
    <span class="size-2 shrink-0 rounded-full {running ? 'animate-pulse bg-amber-500' : 'bg-emerald-500'}" aria-label={running ? 'Working' : 'Ready'}></span>
    {#if onClose}<Button variant="ghost" size="icon-sm" aria-label="Close assistant" onclick={onClose}><XIcon class="size-4" /></Button>{/if}
  </header>

  <div bind:this={log} use:measure role="log" aria-live="polite" aria-relevant="additions text" class="relative min-h-0 flex-1 space-y-5 overflow-y-auto overscroll-contain px-5 py-5 text-sm">
    {#each messages as m, i (i)}
      {#if m.role === 'user'}
        <div class="ml-6 rounded-xl rounded-tr-sm bg-muted px-4 py-3"><p class="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">You</p><p class="break-words whitespace-pre-wrap">{m.text}</p></div>
      {:else}
        <div use:anchorReply data-reply class="space-y-3" style:min-height={i === messages.length - 1 ? `${Math.max(180, logHeight - 40)}px` : undefined}>
          <p class="flex items-center gap-2 text-xs font-semibold"><SparklesIcon class="size-3.5" /> Resume assistant</p>
          {#each m.parts as part, j (j)}
            {#if part.kind === 'text'}
              <p class="break-words leading-7 whitespace-pre-wrap">{part.text}</p>
            {:else}
              <p class="rounded-md border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">{part.label}</p>
            {/if}
          {/each}
          {#if running && i === messages.length - 1}<p class="flex items-center gap-2 text-xs text-muted-foreground"><span class="size-1.5 animate-pulse rounded-full bg-current"></span>{m.parts.length ? 'Working on your request…' : 'Reading your request…'}</p>{/if}
        </div>
      {/if}
    {:else}
      <div class="flex min-h-full flex-col justify-center py-6">
        <SparklesIcon class="mb-4 size-7 text-muted-foreground" />
        <h4 class="text-xl font-semibold tracking-tight">What would you like to change?</h4>
        <p class="mt-3 leading-6 text-muted-foreground">Ask me to rewrite, add, remove, or reorganize any part of your resume. I’ll make the changes and update the document here.</p>
        <div class="mt-6 grid gap-2">
          {#if jobTitle}<Button variant="outline" class="h-auto justify-start whitespace-normal py-3 text-left" onclick={() => send(`Tailor my resume for the ${jobTitle} job.`)}>Tailor my resume for this job</Button>{/if}
          <Button variant="outline" class="h-auto justify-start whitespace-normal py-3 text-left" onclick={() => send('Make my summary more concise and impactful.')}>Sharpen my summary</Button>
          <Button variant="outline" class="h-auto justify-start whitespace-normal py-3 text-left" onclick={() => send('Improve the wording of my experience bullets, preserving all facts.')}>Strengthen my experience bullets</Button>
        </div>
      </div>
    {/each}
  </div>

  {#if notice}<p role="status" class="shrink-0 border-t bg-muted/30 px-4 py-3 text-xs leading-5 text-muted-foreground">{notice}</p>{/if}
  {#if error}<div class="shrink-0 px-4 pb-3"><ErrorBanner message={error} /></div>{/if}
  <div class="shrink-0 border-t bg-muted/15 p-3">
    <div class="rounded-xl border bg-background p-2 focus-within:ring-2 focus-within:ring-ring/20">
      <textarea aria-label="Message" placeholder="Tell me what to change…" rows="3" bind:value={draft} onkeydown={onKey} disabled={running} class="max-h-40 min-h-20 w-full resize-none border-0 bg-transparent p-2 text-sm leading-6 outline-none placeholder:text-muted-foreground disabled:opacity-60"></textarea>
      <div class="flex items-center justify-between gap-2 px-1 pb-1"><span class="text-[10px] text-muted-foreground">Enter to send · Shift + Enter for a new line</span>{#if running}<Button variant="outline" onclick={stop}>Stop</Button>{:else}<Button disabled={!draft.trim()} onclick={() => send(draft)}>Send</Button>{/if}</div>
    </div>
  </div>
</section>
