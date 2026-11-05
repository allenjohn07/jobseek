<script lang="ts">
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import DatePicker from '$lib/components/DatePicker.svelte'
  import StatusSelect from '$lib/components/StatusSelect.svelte'
  import BuildingIcon from '@lucide/svelte/icons/building-2'
  import CircleDotIcon from '@lucide/svelte/icons/circle-dot'
  import CalendarIcon from '@lucide/svelte/icons/calendar'
  import LinkIcon from '@lucide/svelte/icons/link'
  import type { JobInput } from '$lib/types'
  import { normalizeUrl } from '$lib/url'

  let {
    open = $bindable(false),
    onSave,
  }: {
    open?: boolean
    onSave: (input: JobInput) => Promise<void>
  } = $props()

  const empty: JobInput = { company: '', title: '', url: '', status: 'wishlist', appliedOn: '', notes: '' }
  let form = $state<JobInput>({ ...empty })
  let saving = $state(false)
  let error = $state('')

  // Reset the form each time the dialog opens.
  $effect(() => {
    if (open) {
      form = { ...empty }
      error = ''
    }
  })

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    saving = true
    error = ''
    try {
      await onSave({ ...$state.snapshot(form), url: normalizeUrl(form.url) })
      open = false
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      saving = false
    }
  }

  // Borderless, Notion-like field: looks like text until hovered/focused.
  const field =
    'w-full min-w-0 rounded-sm border-0 bg-transparent px-2 py-1 text-sm outline-none placeholder:text-muted-foreground/50 hover:bg-accent/60 focus:bg-accent'
  const row = 'grid grid-cols-[7.25rem_1fr] items-center gap-2'
  const rowLabel = 'flex items-center whitespace-nowrap gap-2 px-2 text-sm text-muted-foreground [&>svg]:size-4'
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="gap-0 overflow-hidden p-0 sm:max-w-md" showCloseButton={false}>
    <Dialog.Title class="sr-only">Add job</Dialog.Title>
    <Dialog.Description class="sr-only">Add a new job application to your tracker.</Dialog.Description>

    <form onsubmit={submit}>
      <!-- Big title, like a Notion page -->
      <div class="px-3 pt-4 pb-1">
        <input
          required
          aria-label="Job title"
          placeholder="Job title"
          bind:value={form.title}
          class="w-full rounded-sm border-0 bg-transparent px-2 py-0.5 text-2xl font-bold tracking-tight outline-none placeholder:text-muted-foreground/40"
        />
      </div>

      <!-- Properties -->
      <div class="grid px-3 pb-2">
        <div class={row}>
          <span class={rowLabel}><BuildingIcon />Company</span>
          <input required aria-label="Company" placeholder="Empty" bind:value={form.company} class={field} />
        </div>
        <div class={row}>
          <span class={rowLabel}><CircleDotIcon />Status</span>
          <div><StatusSelect status={form.status} onChange={(s) => (form.status = s)} /></div>
        </div>
        <div class={row}>
          <span class={rowLabel}><CalendarIcon />Applied on</span>
          <div>
            <DatePicker
              label="Applied on"
              placeholder="Empty"
              value={form.appliedOn}
              onSave={(v) => (form.appliedOn = v)}
              alwaysShowPlaceholder
            />
          </div>
        </div>
        <div class={row}>
          <span class={rowLabel}><LinkIcon />URL</span>
          <input type="text" inputmode="url" autocapitalize="off" spellcheck="false" aria-label="URL" placeholder="Empty" bind:value={form.url} class={field} />
        </div>
      </div>

      <!-- Notes -->
      <div class="border-t px-3 py-2">
        <textarea
          aria-label="Notes"
          placeholder="Add notes…"
          bind:value={form.notes}
          class="field-sizing-content max-h-40 min-h-12 w-full resize-none [scrollbar-width:none] [&::-webkit-scrollbar]:hidden rounded-sm border-0 bg-transparent px-2 py-1 text-sm outline-none placeholder:text-muted-foreground/50 hover:bg-accent/40 focus:bg-accent/60"
        ></textarea>
      </div>

      {#if error}<p class="px-4 pb-2 text-sm text-[#d44c47]">{error}</p>{/if}

      <div class="flex items-center justify-end gap-1 border-t bg-muted/40 px-3 py-2">
        <Button type="button" variant="ghost" onclick={() => (open = false)}>Cancel</Button>
        <Button type="submit" disabled={saving}>{saving ? 'Adding…' : 'Add job'}</Button>
      </div>
    </form>
  </Dialog.Content>
</Dialog.Root>
