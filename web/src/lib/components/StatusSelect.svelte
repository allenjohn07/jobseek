<script lang="ts">
  import * as Select from '$lib/components/ui/select'
  import StatusBadge from '$lib/components/StatusBadge.svelte'
  import { STATUSES, type Status } from '$lib/types'

  let { status, onChange }: { status: Status; onChange: (s: Status) => void } = $props()
</script>

<Select.Root type="single" value={status} onValueChange={(v) => onChange(v as Status)}>
  <!-- Trigger is just the coloured tag; the chevron from the shadcn trigger is hidden. -->
  <Select.Trigger
    aria-label="Status"
    class="h-auto w-full justify-start rounded-sm border-0 bg-transparent p-1 shadow-none hover:bg-accent/60 dark:bg-transparent [&>svg:last-child]:hidden"
  >
    <StatusBadge {status} />
  </Select.Trigger>
  <Select.Content>
    {#each STATUSES as s (s)}
      <Select.Item value={s} label={s}><StatusBadge status={s} /></Select.Item>
    {/each}
  </Select.Content>
</Select.Root>
