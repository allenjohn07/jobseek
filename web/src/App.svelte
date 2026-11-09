<script lang="ts">
    import logo from "$lib/../assets/dart-logo.png";
    import JobsView from "$lib/components/JobsView.svelte";
    import JobTweakView from "$lib/components/JobTweakView.svelte";

    // Tabs live in the URL hash so reloads and back/forward keep your place. No router needed.
    type Tab = "jobs" | "jobtweak";
    const tabs: { id: Tab; label: string }[] = [
        { id: "jobs", label: "Jobs" },
        { id: "jobtweak", label: "JobTweak" },
    ];
    const fromHash = (): Tab =>
        location.hash === "#/jobtweak" ? "jobtweak" : "jobs";
    let tab = $state<Tab>(fromHash());
</script>

<svelte:window onhashchange={() => (tab = fromHash())} />

<main class="mx-auto max-w-[1600px] px-4 pt-6 pb-16 sm:px-8">
    <header class="flex items-center gap-3">
        <img src={logo} alt="JobSeek logo" class="size-10" />
        <div>
            <div class="flex items-baseline gap-2"><h1 class="text-2xl font-bold tracking-tight">JobSeek</h1><span class="text-xs text-muted-foreground">by Allen</span></div>
            <p class="text-sm text-muted-foreground">Every application, one place.</p>
        </div>
    </header>

    <nav aria-label="Sections" class="mt-5 mb-6 flex gap-1 border-b text-sm">
        {#each tabs as t (t.id)}
            <a
                href="#/{t.id}"
                aria-current={tab === t.id ? "page" : undefined}
                class="-mb-px border-b-2 px-3 py-2 {tab === t.id
                    ? 'border-foreground font-medium'
                    : 'border-transparent text-muted-foreground hover:text-foreground'}"
                >{t.label}</a
            >
        {/each}
    </nav>

    {#if tab === "jobs"}
        <JobsView />
    {:else}
        <JobTweakView />
    {/if}
</main>
