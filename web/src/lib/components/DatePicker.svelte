<script lang="ts">
  import * as Popover from '$lib/components/ui/popover'
  import { Calendar } from '$lib/components/ui/calendar'
  import { Button } from '$lib/components/ui/button'
  import { parseDate, today, getLocalTimeZone, type DateValue } from '@internationalized/date'
  import CalendarIcon from '@lucide/svelte/icons/calendar'

  // value is an ISO date string ("2026-10-30") or "" for no date.
  let {
    value,
    onSave,
    label,
    variant = 'cell',
    placeholder = 'Set date',
    id,
    alwaysShowPlaceholder = false,
  }: {
    value: string
    onSave: (value: string) => void
    label: string
    variant?: 'cell' | 'field' // cell = borderless table cell, field = bordered form input
    placeholder?: string
    id?: string
    alwaysShowPlaceholder?: boolean // show the placeholder even when not hovering a table row
  } = $props()

  let open = $state(false)

  // Bad/legacy values (e.g. typed text) just show as "no date" instead of crashing.
  const parsed = $derived.by((): DateValue | undefined => {
    try {
      return value ? parseDate(value) : undefined
    } catch {
      return undefined
    }
  })

  const pretty = $derived(
    parsed
      ? parsed.toDate(getLocalTimeZone()).toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
          year: 'numeric',
        })
      : '',
  )

  function pick(v: DateValue | undefined) {
    if (!v) return // clicking the selected day again: keep it (use "Clear" to remove)
    onSave(v.toString())
    open = false
  }
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    {id}
    aria-label={label}
    class={variant === 'cell'
      ? 'flex w-full items-center gap-1.5 rounded-sm px-2 py-1.5 text-left hover:bg-accent/60 data-[state=open]:bg-accent'
      : 'flex h-9 w-full items-center gap-2 rounded-control border border-input bg-transparent px-3 text-left text-sm'}
  >
    {#if variant === 'field'}<CalendarIcon class="size-4 text-muted-foreground" />{/if}
    {#if pretty}
      <span class="truncate">{pretty}</span>
    {:else}
      <span
        class="truncate text-muted-foreground/60 {variant === 'cell' && !alwaysShowPlaceholder
          ? 'opacity-0 group-hover:opacity-100'
          : ''}">{placeholder}</span
      >
    {/if}
  </Popover.Trigger>
  <Popover.Content align="start" class="w-auto gap-0 p-0">
    <Calendar
      type="single"
      value={parsed}
      onValueChange={pick}
      captionLayout="dropdown"
    />
    <div class="flex items-center justify-between border-t p-1">
      <Button variant="ghost" size="xs" onclick={() => pick(today(getLocalTimeZone()))}>Today</Button>
      <Button
        variant="ghost"
        size="xs"
        disabled={!value}
        onclick={() => {
          onSave('')
          open = false
        }}>Clear</Button
      >
    </div>
  </Popover.Content>
</Popover.Root>
