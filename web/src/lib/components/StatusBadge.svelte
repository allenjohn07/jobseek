<script lang="ts">
  import type { Status } from '$lib/types'

  let { status }: { status: Status } = $props()

  // Dot colours come from the --color-mark-<status> tokens in app.css.
  const marks: Record<Status, string> = {
    wishlist: 'bg-mark-wishlist',
    applied: 'bg-mark-applied',
    interviewing: 'bg-mark-interviewing',
    offer: 'bg-mark-offer',
    rejected: 'bg-mark-rejected',
  }
</script>

<span class="inline-flex items-center gap-2 text-sm leading-5 text-ink capitalize">
  <span data-mark class="size-2 shrink-0 rounded-chip {marks[status] ?? marks.wishlist}"></span>{status}
</span>
