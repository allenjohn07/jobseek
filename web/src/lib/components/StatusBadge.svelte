<script lang="ts">
  import type { Status } from '$lib/types'

  let { status }: { status: Status } = $props()

  // Colours come from the --tag-* tokens in app.css.
  const styles: Record<Status, { tag: string; dot: string }> = {
    wishlist: { tag: 'bg-tag-wishlist-bg text-tag-wishlist-fg', dot: 'bg-tag-wishlist-dot' },
    applied: { tag: 'bg-tag-applied-bg text-tag-applied-fg', dot: 'bg-tag-applied-dot' },
    interviewing: { tag: 'bg-tag-interviewing-bg text-tag-interviewing-fg', dot: 'bg-tag-interviewing-dot' },
    offer: { tag: 'bg-tag-offer-bg text-tag-offer-fg', dot: 'bg-tag-offer-dot' },
    rejected: { tag: 'bg-tag-rejected-bg text-tag-rejected-fg', dot: 'bg-tag-rejected-dot' },
  }
  const s = $derived(styles[status] ?? styles.wishlist)
</script>

<span class="inline-flex items-center gap-1.5 rounded-sm px-1.5 py-0.5 text-sm leading-5 capitalize {s.tag}">
  <span class="size-1.5 rounded-full {s.dot}"></span>{status}
</span>
