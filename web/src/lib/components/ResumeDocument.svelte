<script lang="ts">
  import ResumeCanvas from './ResumeCanvas.svelte'
  import type { ResumeContent } from '$lib/types'
  let { content, printable = true }: { content: ResumeContent; printable?: boolean } = $props()
</script>

<ResumeCanvas {content} {printable} editable={false} />
