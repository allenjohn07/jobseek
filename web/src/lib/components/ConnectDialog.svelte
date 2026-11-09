<script lang="ts">
  import { getMcpInfo } from '$lib/api'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import ErrorBanner from '$lib/components/ErrorBanner.svelte'
  import type { McpInfo } from '$lib/types'

  let { open = $bindable(false) }: { open?: boolean } = $props()

  let info = $state<McpInfo | null>(null)
  let error = $state('')
  let copied = $state(false)

  // Load each time the dialog opens, so the command is always current.
  $effect(() => {
    if (!open) return
    copied = false
    error = ''
    getMcpInfo()
      .then((i) => (info = i))
      .catch((e) => (error = e instanceof Error ? e.message : String(e)))
  })

  async function copy() {
    if (!info) return
    try {
      await navigator.clipboard.writeText(info.command)
      copied = true
    } catch {
      error = 'Could not copy. Select the command and copy it manually.'
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>Connect an AI</Dialog.Title>
      <Dialog.Description>
        Run this once in a terminal to let Claude Code read your jobs and edit your resumes, including section order.
        Ask for suggestions when you want to review and accept or reject changes before applying them.
      </Dialog.Description>
    </Dialog.Header>
    <ErrorBanner message={error} />
    {#if info}
      <pre class="overflow-x-auto rounded-sm bg-muted p-3 text-xs whitespace-pre-wrap break-all" aria-label="Connect command">{info.command}</pre>
      <p class="text-xs text-muted-foreground">The token is private to this computer. Don't share the command.</p>
      <Dialog.Footer>
        <Button onclick={copy}>{copied ? 'Copied' : 'Copy command'}</Button>
      </Dialog.Footer>
    {/if}
  </Dialog.Content>
</Dialog.Root>
