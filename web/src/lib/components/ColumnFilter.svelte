<script lang="ts" module>
  export type FilterOption = { value: string; label: string; count: number }
</script>

<script lang="ts">
  import * as Popover from '$lib/components/ui/popover'
  import { Checkbox } from '$lib/components/ui/checkbox'
  import ListFilterIcon from '@lucide/svelte/icons/list-filter'
  import type { Snippet } from 'svelte'

  let {
    label,
    options,
    selected,
    onChange,
    optionLabel,
  }: {
    label: string
    options: FilterOption[]
    selected: string[] // empty = no filter (show everything)
    onChange: (values: string[]) => void
    optionLabel?: Snippet<[FilterOption]> // custom rendering, e.g. status tags
  } = $props()

  let query = $state('')
  const searchable = $derived(options.length > 8)
  const shown = $derived(
    query.trim()
      ? options.filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()))
      : options,
  )
  const active = $derived(selected.length > 0)

  const toggle = (value: string) =>
    onChange(selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value])
</script>

<Popover.Root onOpenChange={(o) => { if (!o) query = '' }}>
  <!-- The whole header cell is the trigger: one hover effect for label + icon. -->
  <Popover.Trigger
    aria-label="Filter {label}"
    class="group/head flex h-8 w-full items-center gap-1.5 px-3 text-left text-muted-foreground transition-colors hover:bg-accent/60 hover:text-foreground data-[state=open]:bg-accent/60 data-[state=open]:text-foreground {active
      ? 'text-foreground'
      : ''}"
  >
    {label}
    <span
      class="flex items-center gap-0.5 text-xs transition-opacity {active
        ? 'opacity-100'
        : 'opacity-0 group-hover/head:opacity-100 group-data-[state=open]/head:opacity-100'}"
    >
      <ListFilterIcon class="size-3.5" />
      {#if active}{selected.length}{/if}
    </span>
  </Popover.Trigger>
  <Popover.Content align="start" class="w-60 gap-0 p-1.5">
    {#if searchable}
      <input
        type="text"
        placeholder="Search {label.toLowerCase()}…"
        bind:value={query}
        class="mb-1 w-full rounded-sm border-0 bg-accent/60 px-2 py-1.5 text-sm outline-none placeholder:text-muted-foreground/60"
      />
    {/if}
    <div class="max-h-64 overflow-y-auto">
      {#each shown as opt (opt.value)}
        <button
          type="button"
          onclick={() => toggle(opt.value)}
          class="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
        >
          <Checkbox checked={selected.includes(opt.value)} tabindex={-1} class="pointer-events-none" />
          <span class="min-w-0 flex-1 truncate">
            {#if optionLabel}{@render optionLabel(opt)}{:else}{opt.label}{/if}
          </span>
          <span class="text-xs text-muted-foreground">{opt.count}</span>
        </button>
      {:else}
        <p class="px-2 py-3 text-center text-sm text-muted-foreground">No matches</p>
      {/each}
    </div>
    {#if active}
      <div class="mt-1 border-t pt-1">
        <button
          type="button"
          onclick={() => onChange([])}
          class="w-full rounded-sm px-2 py-1.5 text-left text-sm text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          Clear filter
        </button>
      </div>
    {/if}
  </Popover.Content>
</Popover.Root>
