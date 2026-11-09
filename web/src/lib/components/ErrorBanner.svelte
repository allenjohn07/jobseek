<script lang="ts">
  let { message }: { message: string } = $props()
</script>

{#if message}
  <p role="alert" class="rounded-control bg-danger-soft px-4 py-3 text-sm text-danger-soft-foreground">{message}</p>
{/if}
