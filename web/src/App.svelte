<script lang="ts">
    import { createJob, deleteJob, getJobs, updateJob } from "$lib/api";
    import { STATUSES, type Job, type JobInput, type Status } from "$lib/types";
    import * as Table from "$lib/components/ui/table";
    import * as AlertDialog from "$lib/components/ui/alert-dialog";
    import EditableCell from "$lib/components/EditableCell.svelte";
    import StatusSelect from "$lib/components/StatusSelect.svelte";
    import { Button } from "$lib/components/ui/button";
    import PlusIcon from "@lucide/svelte/icons/plus";
    import Trash2Icon from "@lucide/svelte/icons/trash-2";
    import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
    import logo from "$lib/../assets/dart-logo.png";
    import { hostname, normalizeUrl } from "$lib/url";
    import UrlCell from "$lib/components/UrlCell.svelte";
    import DatePicker from "$lib/components/DatePicker.svelte";
    import ColumnFilter, { type FilterOption } from "$lib/components/ColumnFilter.svelte";
    import StatusBadge from "$lib/components/StatusBadge.svelte";
    import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
    import JobFormDialog from "$lib/components/JobFormDialog.svelte";

    let jobs = $state<Job[]>([]);
    let loading = $state(true);
    let error = $state("");

    let formOpen = $state(false);
    let deleting = $state<Job | null>(null);
    // ids of rows whose notes panel is open
    let expanded = $state<number[]>([]);
    const toggle = (id: number) =>
        (expanded = expanded.includes(id)
            ? expanded.filter((x) => x !== id)
            : [...expanded, id]);

    const toInput = (j: Job): JobInput => ({
        company: j.company,
        title: j.title,
        url: j.url,
        status: j.status,
        appliedOn: j.appliedOn,
        notes: j.notes,
    });
  const msg = (e: unknown) => (e instanceof Error ? e.message : String(e));

    // --- Columns: clicking a column header opens its filter menu ---
    type ColKey = "title" | "company" | "status" | "appliedOn" | "url";
    const columns: { key: ColKey; label: string; w: string }[] = [
        { key: "title", label: "Title", w: "w-[26%]" },
        { key: "company", label: "Company", w: "w-[18%]" },
        { key: "status", label: "Status", w: "w-[16%]" },
        { key: "appliedOn", label: "Applied on", w: "w-[15%]" },
        { key: "url", label: "URL", w: "" },
    ];

    // --- Filtering: pick values from each column's filter menu ---
    // A row is shown if, for every column with a selection, its value is one of the picks.
    let filters = $state<Record<ColKey, string[]>>({
        title: [], company: [], status: [], appliedOn: [], url: [],
    });
    const filterKeys = Object.keys(filters) as ColKey[];
    const filtering = $derived(filterKeys.some((k) => filters[k].length > 0));
    const clearFilters = () => {
        for (const k of filterKeys) filters[k] = [];
    };

    const DATE_BUCKETS: [string, string][] = [
        ["future", "Future"],
        ["week", "Past week"],
        ["month", "Past month"],
        ["quarter", "Past 3 months"],
        ["older", "Older"],
        ["none", "No date"],
    ];
    function dateBucket(d: string): string {
        if (!d) return "none";
        const days = (Date.now() - new Date(d).getTime()) / 86_400_000;
        if (Number.isNaN(days)) return "none";
        if (days < 0) return "future";
        if (days <= 7) return "week";
        if (days <= 30) return "month";
        if (days <= 90) return "quarter";
        return "older";
    }

    // The value a job has for a column, as used by that column's filter menu.
    function filterValue(j: Job, key: ColKey): string {
        if (key === "appliedOn") return dateBucket(j.appliedOn);
        if (key === "url") return j.url ? hostname(j.url) : "";
        return j[key];
    }

    // Menu options come from ALL jobs (not the filtered ones) so picks never vanish.
    const filterOptions = $derived.by(() => {
        const out = {} as Record<ColKey, FilterOption[]>;
        for (const key of filterKeys) {
            const counts = new Map<string, number>();
            for (const j of jobs) {
                const v = filterValue(j, key);
                counts.set(v, (counts.get(v) ?? 0) + 1);
            }
            const fixed =
                key === "status"
                    ? STATUSES.map((v) => [v, v] as [string, string])
                    : key === "appliedOn"
                      ? DATE_BUCKETS
                      : null;
            out[key] = fixed
                ? fixed
                      .filter(([v]) => counts.has(v))
                      .map(([v, label]) => ({ value: v, label, count: counts.get(v)! }))
                : [...counts]
                      .sort(([a], [b]) => a.localeCompare(b, undefined, { sensitivity: "base" }))
                      .map(([v, count]) => ({ value: v, label: v || "(none)", count }));
        }
        return out;
    });

    const filteredJobs = $derived(
        jobs.filter((j) =>
            filterKeys.every(
                (k) => filters[k].length === 0 || filters[k].includes(filterValue(j, k)),
            ),
        ),
    );

    async function load() {
        try {
            jobs = await getJobs();
        } catch (e) {
            error = msg(e);
        } finally {
            loading = false;
        }
    }

    function openAdd() {
        formOpen = true;
    }

    // Used by the "add" dialog; it shows any thrown error inside the dialog.
    async function save(input: JobInput) {
        await createJob(input);
        await load(); // POST response lacks createdAt, so just reload
    }

    // Inline edit: update the cell immediately, roll back if the API fails.
    // PUT is a full replace on the server, so we send the whole job each time.
    async function patch(job: Job, changes: Partial<JobInput>) {
        const previous = toInput(job);
        Object.assign(job, changes);
        try {
            await updateJob(job.id, toInput(job));
            error = "";
        } catch (e) {
            Object.assign(job, previous);
            error = msg(e);
        }
    }

    async function confirmDelete() {
        if (!deleting) return;
        const job = deleting;
        try {
            await deleteJob(job.id);
            jobs = jobs.filter((j) => j.id !== job.id);
            error = "";
        } catch (e) {
            error = msg(e);
        }
        deleting = null;
    }

    load();
</script>

<main class="mx-auto max-w-6xl px-6 pt-20 pb-24 sm:px-12">
    <img src={logo} alt="JobSeek logo" class="mb-2 size-16" />
    <div class="flex ">
        <h1 class="mt-4 text-4xl font-bold tracking-tight">JobSeek</h1>
        <h4 class="text-sm text-muted-foreground tracking-tight">by Allen</h4>
    </div>
    <p class="mt-2 text-muted-foreground">
        Every application, one place.
    </p>

    <!-- Notion-style view toolbar -->
    <div class="mt-8 mb-2 flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm">
            <span class="rounded-sm bg-accent px-2 py-1 font-medium">Table</span
            >
            <span class="text-muted-foreground"
                >{filtering
                    ? `${filteredJobs.length} of ${jobs.length} jobs`
                    : `${jobs.length} ${jobs.length === 1 ? "job" : "jobs"}`}</span
            >
            {#if filtering}
                <button
                    type="button"
                    onclick={clearFilters}
                    class="rounded-sm px-1.5 py-0.5 text-muted-foreground underline underline-offset-2 hover:text-foreground"
                    >Clear filters</button
                >
            {/if}
        </div>
        <Button onclick={openAdd}>
            <PlusIcon class="size-3.5" /> New
        </Button>
    </div>

    {#if error}
        <p class="mt-4 rounded-sm bg-[#ffe2dd] p-3 text-sm text-[#5d1715]">
            {error}
        </p>
    {/if}

    {#if loading}
        <p class="mt-6 text-sm text-muted-foreground">Loading…</p>
    {:else}
        <div class="overflow-hidden rounded-md border">
        <Table.Root class="text-sm">
            <Table.Header>
                <Table.Row class="hover:bg-transparent">
                    <Table.Head class="h-8 w-8"
                        ><span class="sr-only">Notes</span></Table.Head
                    >
                    {#each columns as col (col.key)}
                        <Table.Head
                            class="h-8 p-0 font-normal {col.w} {col.key !== 'url' ? 'border-r' : ''}"
                        >
                            <ColumnFilter
                                label={col.label}
                                options={filterOptions[col.key]}
                                selected={filters[col.key]}
                                onChange={(v) => (filters[col.key] = v)}
                                optionLabel={col.key === "status"
                                    ? statusOption
                                    : undefined}
                            />
                        </Table.Head>
                    {/each}
                    <Table.Head class="h-8 w-10"
                        ><span class="sr-only">Delete</span></Table.Head
                    >
                </Table.Row>
            </Table.Header>
            <Table.Body>
                {#each filteredJobs as job (job.id)}
                    <Table.Row class="group border-b-0">
                        <Table.Cell class="p-1">
                            <button
                                type="button"
                                aria-label="{expanded.includes(job.id)
                                    ? 'Hide'
                                    : 'Show'} notes for {job.title}"
                                aria-expanded={expanded.includes(job.id)}
                                onclick={() => toggle(job.id)}
                                class="rounded-sm p-1.5 text-muted-foreground hover:bg-accent"
                            >
                                <ChevronRightIcon
                                    class="size-4 transition-transform {expanded.includes(
                                        job.id,
                                    )
                                        ? 'rotate-90'
                                        : ''}"
                                />
                            </button>
                        </Table.Cell>
                        <Table.Cell class="border-r p-1">
                            <EditableCell
                                label="Title"
                                required
                                class="font-medium"
                                value={job.title}
                                onSave={(v) => patch(job, { title: v })}
                            />
                        </Table.Cell>
                        <Table.Cell class="border-r p-1">
                            <EditableCell
                                label="Company"
                                required
                                value={job.company}
                                onSave={(v) => patch(job, { company: v })}
                            />
                        </Table.Cell>
                        <Table.Cell class="border-r p-1">
                            <StatusSelect
                                status={job.status}
                                onChange={(s) => patch(job, { status: s })}
                            />
                        </Table.Cell>
                        <Table.Cell class="border-r p-1">
                            <DatePicker
                                label="Applied on"
                                value={job.appliedOn}
                                onSave={(v) => patch(job, { appliedOn: v })}
                            />
                        </Table.Cell>
                        <Table.Cell class="p-1">
                            <div class="flex items-center">
                                <UrlCell
                                    value={job.url}
                                    display={hostname}
                                    onSave={(v) => patch(job, { url: v })}
                                />
                                {#if job.url}
                                    <a
                                        href={normalizeUrl(job.url)}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        aria-label="Open {job.title} posting"
                                        class="shrink-0 rounded-sm p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
                                    >
                                        <ExternalLinkIcon class="size-3.5" />
                                    </a>
                                {/if}
                            </div>
                        </Table.Cell>
                        <Table.Cell class="p-1 text-right">
                            <button
                                type="button"
                                aria-label="Delete {job.title}"
                                onclick={() => (deleting = job)}
                                class="rounded-sm p-1.5 text-muted-foreground hover:bg-[#ffe2dd] hover:text-[#d44c47]"
                            >
                                <Trash2Icon class="size-4" />
                            </button>
                        </Table.Cell>
                    </Table.Row>
                    <!-- Always rendered; the grid-rows trick animates height both ways. -->
                    <Table.Row class="hover:bg-transparent">
                        <Table.Cell colspan={7} class="p-0">
                            <div
                                class="grid transition-[grid-template-rows] duration-200 ease-out {expanded.includes(
                                    job.id,
                                )
                                    ? 'grid-rows-[1fr]'
                                    : 'grid-rows-[0fr]'}"
                            >
                                <div
                                    class="overflow-hidden"
                                    inert={!expanded.includes(job.id)}
                                >
                                    <div class="ml-9 pb-5">
                                        <p
                                            class="px-2 pb-1 text-xs text-muted-foreground"
                                        >
                                            Notes
                                        </p>
                                        <EditableCell
                                            multiline
                                            label="Notes for {job.title}"
                                            placeholder="Add notes…"
                                            value={job.notes}
                                            onSave={(v) =>
                                                patch(job, { notes: v })}
                                        />
                                    </div>
                                </div>
                            </div>
                        </Table.Cell>
                    </Table.Row>
                {/each}
                {#if filteredJobs.length === 0}
                    <Table.Row class="hover:bg-transparent">
                        <Table.Cell
                            colspan={7}
                            class="h-20 text-center text-muted-foreground"
                        >
                            {filtering
                                ? "No jobs match these filters."
                                : "No jobs yet. Click “New” to add one."}
                        </Table.Cell>
                    </Table.Row>
                {/if}
            </Table.Body>
        </Table.Root>
        </div>
    {/if}
</main>

{#snippet statusOption(o: FilterOption)}
    <StatusBadge status={o.value as Status} />
{/snippet}

<JobFormDialog bind:open={formOpen} onSave={save} />

<AlertDialog.Root
    open={deleting !== null}
    onOpenChange={(o) => {
        if (!o) deleting = null;
    }}
>
    <AlertDialog.Content>
        <AlertDialog.Header>
            <AlertDialog.Title>Delete this job?</AlertDialog.Title>
            <AlertDialog.Description>
                “{deleting?.title}” at {deleting?.company} will be permanently removed.
            </AlertDialog.Description>
        </AlertDialog.Header>
        <AlertDialog.Footer>
            <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
            <AlertDialog.Action onclick={confirmDelete}
                >Delete</AlertDialog.Action
            >
        </AlertDialog.Footer>
    </AlertDialog.Content>
</AlertDialog.Root>
