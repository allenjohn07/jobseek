<script lang="ts">
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import CheckIcon from "@lucide/svelte/icons/check";

	// Compact, borderless dropdown for the calendar's month/year pickers.
	// Rendered inline (not in a portal) so it works inside the date popover.
	let {
		value,
		options,
		label,
		onSelect,
	}: {
		value: number
		options: { value: number; label: string }[]
		label: string
		onSelect: (value: number) => void
	} = $props();

	let open = $state(false);
	let root = $state<HTMLElement>();
	let list = $state<HTMLElement>();

	const current = $derived(options.find((o) => o.value === value)?.label ?? String(value));

	// Close when clicking anywhere outside this dropdown.
	function outside(e: PointerEvent) {
		if (open && root && !root.contains(e.target as Node)) open = false;
	}

	// Centre the selected option when opening. We set the list's own scrollTop
	// (scrollIntoView would also scroll the whole page, making it jump).
	$effect(() => {
		if (!open || !list) return;
		const sel = list.querySelector<HTMLElement>("[aria-selected=true]");
		if (sel) list.scrollTop = sel.offsetTop - list.clientHeight / 2 + sel.offsetHeight / 2;
	});

	function pick(v: number) {
		open = false;
		if (v !== value) onSelect(v);
	}
</script>

<svelte:window onpointerdown={outside} />

<div class="relative" bind:this={root}>
	<button
		type="button"
		aria-label={label}
		aria-expanded={open}
		onclick={() => (open = !open)}
		class="flex h-(--cell-size) items-center gap-1 rounded-sm px-2 text-xs font-medium select-none hover:bg-accent/60 {open
			? 'bg-accent/60'
			: ''}"
	>
		{current}
		<ChevronDownIcon class="size-3.5 text-muted-foreground" />
	</button>
	{#if open}
		<div
			bind:this={list}
			role="listbox"
			aria-label={label}
			class="absolute top-full left-1/2 z-50 mt-1 max-h-44 min-w-16 -translate-x-1/2 overflow-y-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
		>
			{#each options as o (o.value)}
				<button
					type="button"
					role="option"
					aria-selected={o.value === value}
					onclick={() => pick(o.value)}
					class="flex w-full items-center justify-between gap-2 rounded-sm px-2 py-1 text-left text-xs hover:bg-accent {o.value ===
					value
						? 'bg-accent/60'
						: ''}"
				>
					{o.label}
					{#if o.value === value}<CheckIcon class="size-3" />{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
