<script lang="ts">
  let { message }: { message: string } = $props()
</script>

{#if message}
  <p role="alert" class="rounded-sm bg-danger-soft p-3 text-sm text-danger-soft-foreground">{message}</p>
{/if}
