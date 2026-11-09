<script lang="ts">
    import logo from "$lib/../assets/dart-logo.png";
    import Hero from "$lib/components/Hero.svelte";
    import Segmented from "$lib/components/Segmented.svelte";
    import ThemeSwitch from "$lib/components/ThemeSwitch.svelte";
    import JobsView from "$lib/components/JobsView.svelte";
    import JobTweakView from "$lib/components/JobTweakView.svelte";

    // Tabs live in the URL hash so reloads and back/forward keep your place. No router needed.
    type Tab = "jobs" | "tweak";
    const tabs = [
        { id: "jobs", label: "Jobs", href: "#/jobs" },
        { id: "tweak", label: "Tweak", href: "#/tweak" },
    ];
    // "#/jobtweak" was the old name; keep old links working.
    const fromHash = (): Tab =>
        location.hash === "#/tweak" || location.hash === "#/jobtweak" ? "tweak" : "jobs";
    let tab = $state<Tab>(fromHash());
</script>

<svelte:window onhashchange={() => (tab = fromHash())} />

<main class="mx-auto max-w-[1600px] px-4 pt-5 pb-16 sm:px-8">
    <Hero>
        <div class="flex flex-wrap items-center justify-between gap-6 px-6 py-8 sm:px-10 sm:py-10">
            <div class="flex items-center gap-4">
                <img src={logo} alt="JobSeek logo" class="size-14 rounded-control bg-surface/90 p-1.5 shadow-card" />
                <div>
                    <div class="flex items-baseline gap-3"><h1 class="numeral text-5xl leading-none font-medium">JobSeek</h1><span class="hand text-lg text-on-sky/80">by Allen</span></div>
                    <p class="hand mt-2 text-xl text-on-sky/90">Every application, one place.</p>
                </div>
            </div>
            <div class="flex flex-wrap items-center gap-3">
                <nav aria-label="Sections">
                    <Segmented variant="glass" label="Sections" items={tabs} value={tab} />
                </nav>
                <ThemeSwitch />
            </div>
        </div>
    </Hero>

    <div class="mt-8">
        {#if tab === "jobs"}
            <JobsView />
        {:else}
            <JobTweakView />
        {/if}
    </div>
</main>
