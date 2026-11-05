<!-- lang="ts" turns on TypeScript inside this component -->
<script lang="ts">
  import { getJobs } from './lib/api'      // no file extension needed for .ts imports
  import type { Job } from './lib/types'

  // $state<Job[]>([]) = a reactive variable that holds an array of Jobs, starting empty.
  // The <Job[]> part tells TypeScript what goes in it. Without it, TS would guess "array of nothing".
  let jobs = $state<Job[]>([])
  let loading = $state(true)   // TS infers boolean from the starting value, so no annotation needed
  let error = $state('')       // inferred as string

  async function load() {
    try {
      jobs = await getJobs()
    } catch (e) {
      // In TS, a caught error has type "unknown", because anything can be thrown in JS.
      // So we check: if it's a real Error object, use its message; otherwise convert it to text.
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  load()
</script>

<main class="mx-auto max-w-4xl p-8">
  <h1 class="mb-6 text-3xl font-bold text-gray-900">Job Tracker</h1>

  {#if loading}
    <p class="text-gray-500">Loading…</p>
  {:else if error}
    <p class="rounded-lg bg-red-50 p-4 text-red-700">{error}</p>
  {:else if jobs.length === 0}
    <p class="text-gray-500">No jobs yet. Add one with curl for now!</p>
  {:else}
    <ul class="space-y-3">
      <!-- Hover over "job" in VS Code: it now knows it's a Job, and autocompletes its fields -->
      {#each jobs as job (job.id)}
        <li class="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div>
            <p class="font-semibold text-gray-900">{job.title}</p>
            <p class="text-sm text-gray-500">{job.company}</p>
          </div>
          <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700">
            {job.status}
          </span>
        </li>
      {/each}
    </ul>
  {/if}
</main>
