<script lang="ts">
  import { onDestroy, tick } from 'svelte'
  import { Dialog } from 'bits-ui'
  import ResumeUpload from './ResumeUpload.svelte'
  import { hasContent } from '$lib/resume'
  import {
    streamChat, acceptProposal, createJobResume, getBaseResume, getJobResume, getJobs, getProposals,
    rejectProposal, updateJob, updateResume,
  } from '$lib/api'
  import { toInput } from '$lib/job'
  import type { Job, JobInput, Proposal, Resume, ResumeContent } from '$lib/types'
  import { Button } from '$lib/components/ui/button'
  import EditableCell from '$lib/components/EditableCell.svelte'
  import ErrorBanner from '$lib/components/ErrorBanner.svelte'
  import ResumeCanvas from '$lib/components/ResumeCanvas.svelte'
  import ResumeDocument from '$lib/components/ResumeDocument.svelte'
  import ProposalCard from '$lib/components/ProposalCard.svelte'
  import ConnectDialog from '$lib/components/ConnectDialog.svelte'
  import ChatPanel from '$lib/components/ChatPanel.svelte'
  import Segmented from '$lib/components/Segmented.svelte'
  import PrinterIcon from '@lucide/svelte/icons/printer'
  import PlugIcon from '@lucide/svelte/icons/plug'
  import SparklesIcon from '@lucide/svelte/icons/sparkles'

  const msg = (e: unknown) => (e instanceof Error ? e.message : String(e))

  let jobs = $state<Job[]>([])
  // null = the base resume, otherwise the id of the selected job
  let selected = $state<number | null>(null)
  let resume = $state<Resume | null>(null)
  let missing = $state(false) // selected job has no tailored resume yet
  let loading = $state(true)
  let creating = $state(false)
  let jobSaving = $state(false)
  let error = $state('')
  let mode = $state<'edit' | 'preview'>('edit')
  let canvas = $state<ResumeCanvas>()
  async function flushDocument() { await canvas?.flush() }
  async function openImport() { try { await flushDocument(); importing = true } catch { return } }
  async function setMode(next: 'edit' | 'preview') {
    try { await flushDocument(); mode = next } catch { /* The editor keeps unsaved changes for retry. */ }
  }
  let base = $state<Resume | null>(null)
  let importing = $state(false)
  let search = $state('')
  let generating = $state(false)
  let aiStatus = $state('')
  let generation: AbortController | null = null
  onDestroy(() => generation?.abort())
  const baseReady = $derived(!!base && hasContent(base.content))
  const filteredJobs = $derived(jobs.filter(j => `${j.title} ${j.company}`.toLowerCase().includes(search.toLowerCase())))
  async function saveImport(content: ResumeContent) {
    const token = loadToken
    const target = base ?? await getBaseResume()
    base = await updateResume(target.id, content)
    if (token === loadToken) { importing = false; await select(null) }
  }
  async function tailor() {
    try { await flushDocument() } catch { return }
    if (!job || !baseReady || !job.description.trim() || generating || jobSaving) return
    const id = job.id
    const token = loadToken
    generation = new AbortController()
    const signal = generation.signal
    generating = true; aiStatus = 'Reading your resume and job description…'; error = ''
    try {
      const target = resume ?? await createJobResume(id)
      if (token !== loadToken || signal.aborted) return
      resume = target; missing = false
      await streamChat(target.id, 'Tailor this resume to the saved job description. Read the job and resume, then propose specific improvements using only existing facts. Do not apply changes.', ev => {
        if (token !== loadToken) return
        if (ev.type === 'error') error = ev.message
        if (ev.type === 'text') aiStatus = ev.text
        if (ev.type === 'tool' && ev.name === 'propose_edit') { aiStatus = 'Preparing suggestions for your review…'; loadProposals(target.id) }
      }, signal)
      if (token === loadToken) { await loadProposals(target.id); aiStatus = signal.aborted ? 'Generation stopped. You can review suggestions already created.' : 'Ready to review. Accept the changes you want to keep.' }
    } catch (e) { if (token === loadToken && !signal.aborted) error = msg(e) }
    finally { if (token === loadToken) generating = false }
  }
  async function exportPDF() { try { await flushDocument(); mode = 'preview'; await tick(); window.print() } catch { return } }
  let proposals = $state<Proposal[]>([])
  let resolving = $state<number | null>(null) // proposal being accepted/rejected
  let connectOpen = $state(false)
  let chatOpen = $state(false)

  const job = $derived(jobs.find((j) => j.id === selected) ?? null)

  // Guards against a slow response for a previous selection overwriting the current one.
  let loadToken = 0

  const fetchResume = (id: number | null) => (id === null ? getBaseResume() : getJobResume(id))

  async function select(id: number | null) {
    try { await flushDocument() } catch { return }
    chatOpen = false
    generation?.abort(); generating = false; aiStatus = ''; importing = false
    selected = id
    const token = ++loadToken
    loading = true
    missing = false
    resume = null
    proposals = []
    try {
      const r = await fetchResume(id)
      if (token !== loadToken) return
      resume = r
      if (id === null) base = r
      missing = r === null
      error = ''
    } catch (e) {
      if (token === loadToken) error = msg(e)
    } finally {
      if (token === loadToken) loading = false
    }
  }

  async function init() {
    try {
      jobs = await getJobs()
    } catch (e) {
      error = msg(e)
    }
    await select(null)
  }

  async function createForJob() {
    if (selected === null) return
    const id = selected
    const token = loadToken
    creating = true
    try {
      const created = await createJobResume(id)
      if (token !== loadToken) return
      resume = created
      missing = false
      error = ''
    } catch (e) {
      if (token === loadToken) error = msg(e)
    } finally {
      creating = false
    }
  }

  // Show the change immediately; roll back if the API rejects it.
  async function change(next: ResumeContent) {
    if (!resume) return
    const target = resume
    const previous = target.content
    target.content = next
    try {
      const saved = await updateResume(target.id, next)
      if (resume === target) resume = saved
      if (target.jobId === null) base = saved
      error = ''
    } catch (e) {
      if (resume === target) target.content = previous
      error = msg(e)
      throw e
    }
  }

  // Same full-replace + rollback pattern as the jobs table.
  async function patchJob(j: Job, changes: Partial<JobInput>) {
    const previous = toInput(j)
    Object.assign(j, changes)
    jobSaving = true
    try {
      await updateJob(j.id, toInput(j))
      error = ''
    } catch (e) {
      Object.assign(j, previous)
      error = msg(e)
    } finally { jobSaving = false }
  }

  // --- AI suggestions ---
  const resumeId = $derived(resume?.id ?? null)
  const pending = $derived(proposals.filter((p) => p.status === 'pending'))

  async function refreshAfterAI() {
    const token = loadToken
    const target = resume
    if (!target) return
    await loadProposals(target.id)
    try {
      const fresh = await fetchResume(selected)
      if (token === loadToken && resume === target && fresh) {
        resume = fresh
        if (fresh.jobId === null) base = fresh
      }
    } catch (e) { if (token === loadToken) error = msg(e) }
  }

  async function loadProposals(id: number) {
    try {
      const list = await getProposals(id)
      if (id === resumeId) proposals = list
    } catch {
      // A failed background refresh shouldn't interrupt editing; the next poll retries.
    }
  }

  // Suggestions arrive from outside this page (e.g. Claude Code), so poll while a resume is open.
  $effect(() => {
    const id = resumeId
    if (id === null) return
    loadProposals(id)
    const timer = setInterval(() => loadProposals(id), 4000)
    return () => clearInterval(timer)
  })

  async function resolve(p: Proposal, accept: boolean) {
    try { await flushDocument() } catch { return }
    resolving = p.id
    const selection = selected
    const token = loadToken
    try {
      await (accept ? acceptProposal(p.id) : rejectProposal(p.id))
      error = ''
      if (accept) {
        // The server changed the resume, so show its version.
        const fresh = await fetchResume(selection)
        if (token === loadToken && fresh && resume && fresh.id === resume.id) resume = fresh
      }
    } catch (e) {
      error = msg(e)
    } finally {
      resolving = null
      await loadProposals(p.resumeId)
    }
  }


  const title = $derived(job ? `${job.title} · ${job.company}` : 'Base resume')
  const navBtn = (active: boolean) =>
    `w-full truncate rounded-control px-3 py-2 text-left text-sm transition-colors duration-(--duration-fast) hover:bg-sunken ${active ? 'bg-sunken font-medium text-ink' : 'text-ink-muted'}`

  init()
</script>

<Dialog.Root bind:open={chatOpen}>
<div class="mb-6 flex flex-wrap items-end justify-between gap-3">
  <div><p class="hand text-xl text-ink-muted">Tweak</p><h2 class="mt-1 text-3xl font-semibold">Your experience. The right emphasis.</h2><p class="mt-2 text-sm text-muted-foreground">One base resume. A considered version for every opportunity.</p></div>
  <Button variant="outline" onclick={() => connectOpen = true}><PlugIcon class="size-4" /> Connect AI</Button>
</div>
<div class="grid items-start gap-5 lg:grid-cols-[13rem_minmax(0,1fr)]">
  <nav aria-label="Resumes" class="rounded-card bg-card p-3 shadow-card lg:sticky lg:top-5">
    <button type="button" class="{navBtn(selected === null)} py-3" aria-current={selected === null} onclick={() => select(null)}>Base resume<span class="mt-1 block text-xs font-normal text-muted-foreground">{baseReady ? 'Your saved starting point' : 'Upload to get started'}</span></button>
    <p class="hand mt-6 mb-2 px-2 text-lg text-ink-muted">Your opportunities</p>
    <input aria-label="Search jobs" bind:value={search} placeholder="Search jobs…" class="mb-2 w-full rounded-control border-0 bg-sunken px-3 py-2 text-sm outline-none placeholder:text-ink-faint focus:ring-2 focus:ring-ring/30" />
    <div class="max-h-96 space-y-1 overflow-y-auto">
    {#each filteredJobs as j (j.id)}
      <button type="button" class={navBtn(selected === j.id)} aria-current={selected === j.id} onclick={() => select(j.id)}>{j.title}<span class="mt-1 block truncate text-xs font-normal text-muted-foreground">{j.company}</span></button>
    {:else}<p class="p-2 text-xs text-muted-foreground">{jobs.length ? 'No matching jobs.' : 'Add an opportunity in the Jobs tab.'}</p>{/each}
    </div>
  </nav>
  <section class="min-w-0">
    {#if error}<div class="mb-4"><ErrorBanner message={error} /></div>{/if}
    {#if importing}
      <ResumeUpload onSave={saveImport} onCancel={() => importing = false} />
    {:else if loading}<p class="p-8 text-sm text-muted-foreground">Loading resume…</p>
    {:else}
      <header class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div><h3 class="text-lg font-semibold">{title}</h3><p class="mt-1 text-xs text-muted-foreground">{job ? 'Tailored copy · Your base resume stays yours' : 'The foundation for every tailored resume'}</p></div>
        <div class="flex flex-wrap items-center gap-2">
          {#if !job}<Button variant="outline" onclick={openImport}>{baseReady ? 'Replace resume' : 'Upload resume'}</Button>{/if}
          {#if resume}
            <Dialog.Trigger>
              {#snippet child({ props })}<Button variant="outline" {...props}><SparklesIcon class="size-4" /> Ask AI</Button>{/snippet}
            </Dialog.Trigger>
            <Segmented label="View" items={[{ id: 'preview', label: 'preview' }, { id: 'edit', label: 'edit' }]} value={mode} onChange={(m) => setMode(m as 'edit' | 'preview')} />
            <Button variant="outline" onclick={exportPDF}><PrinterIcon class="size-4" /> Export PDF</Button>
          {/if}
        </div>
      </header>
      <div class="grid items-start gap-5 {job || pending.length ? 'xl:grid-cols-[minmax(0,1fr)_24rem]' : ''}">
        <div class="min-w-0">
          {#if !job && !baseReady && !resume?.content.document}
            <div class="rounded-card border-2 border-dashed border-line bg-card/60 p-10 text-center"><SparklesIcon class="mx-auto mb-4 size-8 text-primary" /><h3 class="text-xl font-semibold">Start with the resume you already have</h3><p class="mx-auto mt-2 max-w-md text-sm text-muted-foreground">Upload a PDF or Word document, review the content, and save it as your base. Every job gets its own tailored copy.</p><Button class="mt-5" onclick={openImport}>Upload your resume</Button></div>
          {:else if missing && job}
            <div class="rounded-card bg-card p-8 shadow-card"><p class="hand text-lg text-ink-muted">A fresh perspective</p><h3 class="mt-3 text-xl font-semibold">Make your experience speak to this role.</h3><p class="mt-3 text-sm text-muted-foreground">Tailor your saved base resume to {job.company}. AI suggests changes for you to review before anything is applied.</p><Button variant="outline" class="mt-5" disabled={creating || !baseReady} onclick={createForJob}>{creating ? 'Creating…' : 'Create from base resume'}</Button></div>
            {#if base}<div class="mt-4 overflow-hidden rounded-card bg-card shadow-card"><p class="bg-sunken px-5 py-2.5 text-xs text-ink-muted">Base resume preview · Tailor to create this job’s version</p><ResumeDocument content={base.content} /></div>{/if}
          {:else if resume}
            <div class="overflow-hidden rounded-card bg-card shadow-card">{#if mode === 'edit'}<ResumeCanvas bind:this={canvas} content={resume.content} onChange={change} />{:else}<ResumeDocument content={resume.content} />{/if}</div>
          {/if}
        </div>
        {#if job || pending.length}<aside class="min-w-0 space-y-4">
          {#if job}
            <section class="rounded-card bg-card p-5 shadow-card"><p class="hand text-lg text-ink-muted">01 · The opportunity</p><h3 class="mt-2 font-semibold">Job description</h3><p class="mt-1 mb-3 text-xs text-muted-foreground">Saved with this job. AI uses this to focus your resume.</p><EditableCell class="max-h-80 overflow-y-auto text-sm" multiline label="Job description" placeholder="Paste the full job description…" value={job.description} onSave={(v) => patchJob(job, { description: v })} /></section>
            <section class="rounded-card bg-sunken p-5"><p class="hand text-lg text-ink-muted">02 · Find the fit</p><h3 class="mt-2 font-semibold">Tailor with AI</h3><p class="mt-2 text-sm text-muted-foreground">Highlight relevant experience using the facts in your resume.</p>
              {#if !baseReady}<Button variant="outline" class="mt-3" onclick={async () => { await select(null); importing = true }}>Upload a base resume</Button>{:else if !job.description.trim()}<p class="mt-3 text-xs text-muted-foreground">Add the job description above to start tailoring.</p>{/if}
              <Button class="mt-4 w-full" disabled={!baseReady || !job.description.trim() || generating || creating || jobSaving} onclick={tailor}><SparklesIcon class="size-4" />{generating ? 'Tailoring…' : 'Tailor resume'}</Button>
              {#if generating}<Button variant="ghost" class="mt-2 w-full" onclick={() => generation?.abort()}>Stop generation</Button>{/if}
              {#if aiStatus}<p role="status" class="mt-3 max-h-40 overflow-y-auto text-xs text-muted-foreground">{aiStatus}</p>{/if}
            </section>
          {/if}
          {#if pending.length}<section aria-label="AI suggestions" class="space-y-3"><h3 class="text-sm font-semibold">AI suggestions ({pending.length})</h3>{#each pending as p (p.id)}<ProposalCard proposal={p} busy={resolving !== null || generating} onAccept={() => resolve(p, true)} onReject={() => resolve(p, false)} />{/each}</section>{/if}

        </aside>{/if}
      </div>
    {/if}
  </section>
</div>
{#if resume}
  {@const sheetResume = resume}
  {#key sheetResume.id}
    <Dialog.Portal>
      <Dialog.Content forceMount trapFocus={false} preventScroll={false} style={chatOpen ? undefined : 'display: none'} onInteractOutside={(event) => event.preventDefault()} class="agent-sheet fixed inset-y-2 right-2 z-50 flex h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-[30rem] flex-col overflow-hidden rounded-sheet bg-surface shadow-lift outline-none">
        {#snippet child({ props })}
        <div {...props} aria-modal="false">
        <Dialog.Title class="sr-only">Resume assistant</Dialog.Title>
        <Dialog.Description class="sr-only">Ask AI to make changes to the current resume. You can keep working on your resume while this panel is open.</Dialog.Description>
        <ChatPanel resumeId={sheetResume.id} jobTitle={job?.title ?? ''} onActivity={refreshAfterAI} beforeSend={flushDocument} fillHeight onClose={() => chatOpen = false} />
        </div>
        {/snippet}
      </Dialog.Content>
    </Dialog.Portal>
  {/key}
{/if}
</Dialog.Root>
<ConnectDialog bind:open={connectOpen} />
<style>
  :global(.agent-sheet[data-state="open"]), :global(.agent-sheet[data-open]) {
    animation: agent-sheet-in var(--duration-slow) var(--ease-out-soft);
  }
  @keyframes agent-sheet-in { from { transform: translateX(100%); } to { transform: translateX(0); } }
  @media (prefers-reduced-motion: reduce) { :global(.agent-sheet) { animation: none !important; } }
</style>
