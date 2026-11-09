<script lang="ts">
  import type { Component } from 'svelte'

  // A small borderless icon button (Notion-style). `danger` gives the destructive hover.
  let {
    icon: Icon,
    label,
    onclick,
    danger = false,
  }: {
    icon: Component<{ class?: string }>
    label: string
    onclick: () => void
    danger?: boolean
  } = $props()
</script>

<button
  type="button"
  aria-label={label}
  title={label}
  {onclick}
  class="shrink-0 rounded-sm p-1.5 text-muted-foreground {danger
    ? 'hover:bg-danger-soft hover:text-danger'
    : 'hover:bg-accent hover:text-foreground'}"
>
  <Icon class="size-4" />
</button>
