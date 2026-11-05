<script lang="ts">
  import * as Popover from '$lib/components/ui/popover'
  import { Button } from '$lib/components/ui/button'
  import ExternalLinkIcon from '@lucide/svelte/icons/external-link'
  import { normalizeUrl } from '$lib/url'

  // Shows a short label (e.g. the domain) in the table; clicking opens a small
  // popover with the full URL, editable, plus an "Open" link.
  let {
    value,
    onSave,
    display,
    label = 'URL',
  }: {
    value: string
    onSave: (value: string) => void
    display: (value: string) => string
    label?: string
  } = $props()

  let open = $state(false)
  let draft = $state('')
  let lastSaved = '' // guards against saving twice (Enter, then the close callback)

  function commit() {
    const next = normalizeUrl(draft) // add https:// if missing
    if (next === lastSaved) return
    lastSaved = next
    onSave(next)
  }

  // Only fires for user-driven changes (click away / Escape), not for `open = false` in code.
  function onOpenChange(o: boolean) {
    if (o) {
      draft = value
      lastSaved = value
    } else commit()
  }
</script>

<Popover.Root bind:open {onOpenChange}>
  <Popover.Trigger
    aria-label={label}
    class="flex w-full min-w-0 items-center rounded-sm px-2 py-1.5 text-left text-muted-foreground hover:bg-accent/60 data-[state=open]:bg-accent"
  >
    {#if value}
      <span class="truncate">{display(value)}</span>
    {:else}
      <span class="truncate text-muted-foreground/60 opacity-0 group-hover:opacity-100">Empty</span>
    {/if}
  </Popover.Trigger>
  <Popover.Content align="start" class="w-auto min-w-56 max-w-[calc(100vw-2rem)] gap-1.5 p-2">
    <input
      type="text"
      inputmode="url"
      autocapitalize="off"
      spellcheck="false"
      aria-label="Full URL"
      placeholder="google.com/careers"
      bind:value={draft}
      onfocus={(e) => e.currentTarget.select()}
      onkeydown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          commit()
          open = false
        }
        if (e.key === 'Escape') draft = value // revert, nothing saved
      }}
      class="field-sizing-content min-w-full max-w-full rounded-sm border-0 bg-accent/60 px-2 py-1.5 text-sm outline-none placeholder:text-muted-foreground/50 focus:bg-accent"
    />
    <div class="flex items-center justify-between">
      <span class="px-1 text-xs text-muted-foreground">Enter to save</span>
      {#if draft.trim()}
        <Button
          variant="ghost"
          size="xs"
          href={normalizeUrl(draft)}
          target="_blank"
          rel="noopener noreferrer"
        >
          Open <ExternalLinkIcon />
        </Button>
      {/if}
    </div>
  </Popover.Content>
</Popover.Root>
