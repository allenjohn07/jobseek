<script lang="ts">
  import type { Status } from '$lib/types'

  let { status }: { status: Status } = $props()

  // Notion's tag colours: soft background + dark text of the same hue.
  const styles: Record<Status, { tag: string; dot: string }> = {
    wishlist: { tag: 'bg-[#e3e2e0] text-[#32302c]', dot: 'bg-[#9b9a97]' },
    applied: { tag: 'bg-[#d3e5ef] text-[#183347]', dot: 'bg-[#5b97bd]' },
    interviewing: { tag: 'bg-[#fdecc8] text-[#402c1b]', dot: 'bg-[#cb912f]' },
    offer: { tag: 'bg-[#dbeddb] text-[#1c3829]', dot: 'bg-[#4f9768]' },
    rejected: { tag: 'bg-[#ffe2dd] text-[#5d1715]', dot: 'bg-[#d44c47]' },
  }
  const s = $derived(styles[status] ?? styles.wishlist)
</script>

<span class="inline-flex items-center gap-1.5 rounded-sm px-1.5 py-0.5 text-sm leading-5 capitalize {s.tag}">
  <span class="size-1.5 rounded-full {s.dot}"></span>{status}
</span>
