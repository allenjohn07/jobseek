<script lang="ts">
  import type { Snippet } from 'svelte'

  // The sky: the app's one "alive" surface (glass controls belong here and nowhere else).
  // A soft gradient with small twinkling dots that lean toward the pointer.
  // With reduced motion the scene is still.
  let { children }: { children: Snippet } = $props()

  let el = $state<HTMLElement>()
  let frame = 0
  const still = () => typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

  function lean(e: PointerEvent) {
    if (!el || still() || frame) return
    const r = el.getBoundingClientRect()
    const x = ((e.clientX - r.left) / r.width) * 2 - 1
    const y = ((e.clientY - r.top) / r.height) * 2 - 1
    frame = requestAnimationFrame(() => {
      frame = 0
      el?.style.setProperty('--px', x.toFixed(3))
      el?.style.setProperty('--py', y.toFixed(3))
    })
  }
  function settle() {
    el?.style.setProperty('--px', '0')
    el?.style.setProperty('--py', '0')
  }

  // Decorative only (not an interaction), so listeners are attached here rather than as element handlers.
  $effect(() => {
    const node = el
    if (!node) return
    node.addEventListener('pointermove', lean)
    node.addEventListener('pointerleave', settle)
    return () => {
      node.removeEventListener('pointermove', lean)
      node.removeEventListener('pointerleave', settle)
      cancelAnimationFrame(frame)
      frame = 0
    }
  })

  // Small dots across the sky: [x%, y%, radius].
  const stars = [
    [8, 22, 1.2], [15, 64, 0.8], [23, 34, 1], [31, 14, 0.7], [38, 52, 1.1], [47, 26, 0.8],
    [54, 70, 0.9], [61, 18, 1.2], [68, 44, 0.7], [44, 8, 0.9], [27, 78, 0.7], [57, 40, 0.6],
    [74, 12, 1], [79, 66, 0.8], [84, 30, 1.1], [90, 74, 0.7], [95, 20, 0.9], [72, 84, 0.8],
  ]
</script>

<section bind:this={el} class="hero relative isolate overflow-hidden rounded-sheet bg-sky text-on-sky shadow-card">
  <div aria-hidden="true" class="pointer-events-none absolute inset-0 -z-10">
    <svg class="layer stars absolute inset-0 size-full" style:--depth="4">
      {#each stars as [x, y, r], i (i)}
        <circle cx="{x}%" cy="{y}%" {r} class="star" style:animation-delay="{(i * 0.7) % 5}s" />
      {/each}
    </svg>

  </div>
  {@render children()}
</section>

<style>
  .hero {
    --px: 0;
    --py: 0;
  }
  .layer {
    translate: calc(var(--px) * var(--depth) * -1px) calc(var(--py) * var(--depth) * -0.5px);
    transition: translate var(--duration-slow) var(--ease-out-soft);
  }
  .star {
    fill: var(--star);
    animation: twinkle 5s ease-in-out infinite;
  }
  @keyframes twinkle {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.35; }
  }
  @media (prefers-reduced-motion: reduce) {
    .layer { transition: none; translate: none; }
    .star { animation: none; }
  }
</style>
