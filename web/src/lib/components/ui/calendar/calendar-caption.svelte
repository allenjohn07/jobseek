<script lang="ts">
	import { DateFormatter, getLocalTimeZone, type DateValue } from "@internationalized/date";
	import CalendarCaptionSelect from "./calendar-caption-select.svelte";
	import type Calendar from "./calendar.svelte";
	import type { ComponentProps } from "svelte";

	let {
		captionLayout,
		months,
		monthFormat,
		years,
		yearFormat,
		month,
		locale,
		placeholder = $bindable(),
		monthIndex = 0,
	}: {
		captionLayout: ComponentProps<typeof Calendar>["captionLayout"];
		months?: unknown;
		monthFormat: "short" | "long" | "narrow" | "numeric" | "2-digit" | ((month: number) => string) | undefined;
		years?: unknown;
		yearFormat: "numeric" | "2-digit" | ((year: number) => string) | undefined;
		month: DateValue;
		placeholder: DateValue | undefined;
		locale: string;
		monthIndex: number;
	} = $props();

	function formatYear(date: DateValue) {
		const dateObj = date.toDate(getLocalTimeZone());
		if (typeof yearFormat === "function") return yearFormat(dateObj.getFullYear());
		return new DateFormatter(locale, { year: yearFormat }).format(dateObj);
	}

	// Month list (1-12) labelled in the calendar's locale.
	const monthOptions = $derived(
		Array.from({ length: 12 }, (_, i) => ({
			value: i + 1,
			label: new DateFormatter(locale, { month: "short" }).format(new Date(2000, i, 1)),
		})),
	);
	// Year list: 1980 .. 15 years ahead, widened if the shown year is outside it.
	const yearOptions = $derived.by(() => {
		const now = new Date().getFullYear();
		const start = Math.min(1980, month.year);
		const end = Math.max(now + 15, month.year);
		return Array.from({ length: end - start + 1 }, (_, i) => ({
			value: start + i,
			label: String(start + i),
		}));
	});

	function formatMonth(date: DateValue) {
		const dateObj = date.toDate(getLocalTimeZone());
		if (typeof monthFormat === "function") return monthFormat(dateObj.getMonth() + 1);
		return new DateFormatter(locale, { month: monthFormat }).format(dateObj);
	}
</script>

{#snippet MonthSelect()}
	<CalendarCaptionSelect
		label="Month"
		value={month.month}
		options={monthOptions}
		onSelect={(v) => {
			if (!placeholder) return;
			placeholder = placeholder.set({ month: v }).subtract({ months: monthIndex });
		}}
	/>
{/snippet}

{#snippet YearSelect()}
	<CalendarCaptionSelect
		label="Year"
		value={month.year}
		options={yearOptions}
		onSelect={(v) => {
			if (!placeholder) return;
			placeholder = placeholder.set({ year: v });
		}}
	/>
{/snippet}

{#if captionLayout === "dropdown"}
	{@render MonthSelect()}
	{@render YearSelect()}
{:else if captionLayout === "dropdown-months"}
	{@render MonthSelect()}
	{#if placeholder}
		{formatYear(placeholder)}
	{/if}
{:else if captionLayout === "dropdown-years"}
	{#if placeholder}
		{formatMonth(placeholder)}
	{/if}
	{@render YearSelect()}
{:else}
	{formatMonth(month)} {formatYear(month)}
{/if}
