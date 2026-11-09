<script lang="ts">
  import { Button } from '$lib/components/ui/button'
  import { toDocument, documentText } from '$lib/resumeDocument'
  import ResumeDocument from './ResumeDocument.svelte'
  import type { ResumeContent, Proposal } from '$lib/types'

  let {
    proposal,
    busy = false,
    onAccept,
    onReject,
  }: {
    proposal: Proposal
    busy?: boolean
    onAccept: () => void
    onReject: () => void
  } = $props()

  const verb = { replace: 'Rewrite', insert_after: 'Add after', delete: 'Remove', replace_document: 'Document changes' } as const
  let reviewing = $state(false)
  function snapshot(raw: string): ResumeContent | null {
    try { const content = JSON.parse(raw) as ResumeContent; return content.basics && Array.isArray(content.experience) && Array.isArray(content.education) && Array.isArray(content.skills) && Array.isArray(content.projects) ? content : null } catch { return null }
  }
  const before = $derived(proposal.op === 'replace_document' ? snapshot(proposal.beforeText) : null)
  const after = $derived(proposal.op === 'replace_document' ? snapshot(proposal.afterText) : null)
  function order(content: ResumeContent) {
    return (toDocument(content).content ?? []).filter(node => node.type === 'heading' && node.attrs?.level === 2).map(documentText).join(' → ') || 'No named sections'
  }
  const removed = 'rounded-control bg-danger-soft px-2.5 py-1.5 text-danger-soft-foreground line-through'
  const added = 'rounded-control bg-success-soft px-2.5 py-1.5 text-success-soft-foreground'
</script>

<div class="animate-rise rounded-card bg-card p-4 text-sm shadow-card">
  <p class="hand text-base text-ink-muted">{verb[proposal.op]}</p>
  <div class="mt-1.5 grid gap-1">
    {#if proposal.op === 'replace_document'}
      {#if before && after}
        <p class="mt-2 text-xs font-medium">Section order</p>
        <p class="text-xs leading-5 text-muted-foreground"><span class="font-medium">Current:</span> {order(before)}</p>
        <p class="text-xs leading-5"><span class="font-medium">Proposed:</span> {order(after)}</p>
        <details bind:open={reviewing} class="mt-2 rounded-control bg-sunken">
          <summary class="cursor-pointer px-3 py-2 text-xs font-medium">Review full document</summary>
          {#if reviewing}
            <div class="max-h-[65vh] space-y-4 overflow-y-auto border-t border-line p-2">
              <div><p class="p-2 text-xs font-semibold">Current resume</p><div class="overflow-hidden rounded-control shadow-card"><ResumeDocument printable={false} content={before} /></div></div>
              <div><p class="p-2 text-xs font-semibold">Proposed resume</p><div class="overflow-hidden rounded-control shadow-card"><ResumeDocument printable={false} content={after} /></div></div>
            </div>
          {/if}
        </details>
      {:else}<p class="text-sm text-muted-foreground">This document suggestion cannot be displayed. Reject it and request a new one.</p>{/if}
    {:else if proposal.op === 'insert_after'}
      <p class="px-1.5 text-muted-foreground">{proposal.beforeText || '(empty)'}</p>
      <p class={added}>{proposal.afterText}</p>
    {:else}
      <p class={removed}>{proposal.beforeText || '(empty)'}</p>
      {#if proposal.op === 'replace'}<p class={added}>{proposal.afterText}</p>{/if}
    {/if}
  </div>
  {#if proposal.reason}<p class="mt-2 text-muted-foreground">{proposal.reason}</p>{/if}
  <div class="mt-3 flex justify-end gap-1">
    <Button variant="ghost" disabled={busy} onclick={onReject}>Reject</Button>
    <Button disabled={busy || (proposal.op === 'replace_document' && (!before || !after))} onclick={onAccept}>Accept</Button>
  </div>
</div>
