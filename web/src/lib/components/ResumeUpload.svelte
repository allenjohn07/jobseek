<script lang="ts">
  import { onDestroy } from 'svelte'
  import { importResume } from '$lib/api'
  import { extractResume } from '$lib/resumeImport'
  import type { ResumeContent } from '$lib/types'
  import ResumeEditor from './ResumeEditor.svelte'
  import ErrorBanner from './ErrorBanner.svelte'
  import { Button } from './ui/button'
  let { onSave, onCancel }: { onSave: (content: ResumeContent) => Promise<void>; onCancel: () => void } = $props()
  let source = $state('')
  let draft = $state<ResumeContent | null>(null)
  let busy = $state(false)
  let saving = $state(false)
  let error = $state('')
  let filename = $state('')
  let input = $state<HTMLInputElement>()
  let controller: AbortController | null = null
  onDestroy(() => controller?.abort())
  async function read(file?: File) {
    if (!file || busy) return
    busy = true; error = ''; draft = null
    try { source = await extractResume(file); filename = file.name }
    catch (e) { error = (e as Error).message }
    finally { busy = false }
  }
  async function organize() {
    busy = true; error = ''; controller = new AbortController()
    try { draft = await importResume(source, controller.signal) }
    catch (e) { if (!controller.signal.aborted) error = (e as Error).message }
    finally { busy = false }
  }
  async function save() {
    if (!draft) return
    saving = true; error = ''
    try { await onSave(draft) } catch (e) { error = (e as Error).message }
    finally { saving = false }
  }
</script>
<section class="rounded-card bg-card p-5 shadow-card sm:p-8" aria-label="Import base resume">
  <p class="hand text-xl text-ink-muted">Your starting point</p>
  <h2 class="mt-2 text-2xl font-semibold">{draft ? 'Review your base resume' : 'Bring your experience with you'}</h2>
  <p class="mt-2 text-sm text-muted-foreground">{draft ? 'Check the imported content before saving. Your job-specific resumes keep their existing content.' : 'Upload your current resume. AI will organize the content into a clean, editable template for you to review.'}</p>
  {#if error}<div class="mt-4"><ErrorBanner message={error} /></div>{/if}
  <div class="mt-6 grid gap-6 {draft ? 'xl:grid-cols-2' : ''}">
    <div>
      {#if !draft}
        <div role="region" aria-label="Resume drop zone" ondragover={(e) => e.preventDefault()} ondrop={(e) => { e.preventDefault(); read(e.dataTransfer?.files[0]) }} class="rounded-card border-2 border-dashed border-line bg-sunken p-8 text-center">
          <p class="mb-3 font-medium">Drop your resume here</p>
          <Button variant="outline" disabled={busy} onclick={() => input?.click()}>Choose resume</Button>
          <input bind:this={input} type="file" aria-label="Upload resume" accept=".pdf,.docx" class="sr-only" disabled={busy} onchange={() => read(input?.files?.[0])} />
          <p class="mt-3 text-xs text-muted-foreground">PDF or DOCX · Up to 10 MB</p>
          {#if filename}<p class="mt-2 text-sm">{filename}</p>{/if}
        </div>
      {/if}
      <label class="mt-4 block text-sm font-medium" for="resume-source">{draft ? 'Original resume text' : 'Or paste your resume text'}</label>
      <textarea id="resume-source" bind:value={source} readonly={draft !== null} disabled={busy} rows={draft ? 22 : 8} class="mt-2 w-full rounded-control border-0 bg-sunken p-3 text-sm outline-none focus:ring-2 focus:ring-ring/30" placeholder="Paste the content of your current resume…"></textarea>
    </div>
    {#if draft}<div class="rounded-card bg-sunken p-4"><ResumeEditor content={draft} onChange={(next) => draft = next} /></div>{/if}
  </div>
  <div class="mt-5 flex flex-wrap justify-end gap-2">
    <Button variant="ghost" disabled={saving} onclick={() => { controller?.abort(); onCancel() }}>Cancel</Button>
    {#if draft}
      <Button variant="outline" disabled={saving} onclick={() => draft = null}>Back to source</Button>
      <Button disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save base resume'}</Button>
    {:else}
      <Button disabled={busy || !source.trim()} onclick={organize}>{busy ? 'Reading resume…' : 'Organize with AI'}</Button>
    {/if}
  </div>
</section>
