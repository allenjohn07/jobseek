<script lang="ts">
  import type { Component } from 'svelte'

  // Segmented control: a track with a thumb that glides to the selected option.
  // Items with an `href` render as links (aria-current), others as toggle buttons (aria-pressed).
  // variant="glass" is only for use over the sky (see Hero.svelte).
  type Item = { id: string; label: string; href?: string; icon?: Component<{ class?: string }> }

  let {
    items,
    value,
    label,
    onChange,
    variant = 'surface',
    iconOnly = false,
  }: {
    items: Item[]
    value: string
    label: string
    onChange?: (id: string) => void
    variant?: 'surface' | 'glass'
    iconOnly?: boolean // show only icons; each item's label is its accessible name
  } = $props()

  const index = $derived(Math.max(0, items.findIndex((i) => i.id === value)))
  const glass = $derived(variant === 'glass')
  const option = (active: boolean) =>
    `relative z-10 flex items-center justify-center gap-1.5 rounded-chip ${iconOnly ? 'px-2.5' : 'px-4'} py-1.5 text-center text-sm font-medium capitalize transition-colors duration-(--duration-base) ${
      active ? 'text-ink' : glass ? 'text-on-sky hover:text-on-sky' : 'text-ink-muted hover:text-ink'
    }`
</script>

{#snippet content(item: Item)}
  {#if item.icon}<item.icon class="size-4" />{/if}
  {#if !iconOnly}{item.label}{/if}
{/snippet}

<div
  role="group"
  aria-label={label}
  class="relative inline-grid rounded-chip p-1 {glass ? 'glass glass-sky' : 'bg-wash'}"
  style:grid-template-columns="repeat({items.length}, minmax(0, 1fr))"
>
  <span
    aria-hidden="true"
    class="absolute inset-y-1 left-1 rounded-chip bg-surface shadow-card transition-transform duration-(--duration-slow) ease-spring"
    style:width="calc((100% - 0.5rem) / {items.length})"
    style:transform="translateX({index * 100}%)"
  ></span>
  {#each items as item (item.id)}
    {#if item.href}
      <a href={item.href} aria-current={item.id === value ? 'page' : undefined} aria-label={iconOnly ? item.label : undefined} title={iconOnly ? item.label : undefined} class={option(item.id === value)}>{@render content(item)}</a>
    {:else}
      <button type="button" aria-pressed={item.id === value} aria-label={iconOnly ? item.label : undefined} title={iconOnly ? item.label : undefined} class={option(item.id === value)} onclick={() => onChange?.(item.id)}>{@render content(item)}</button>
    {/if}
  {/each}
</div>
