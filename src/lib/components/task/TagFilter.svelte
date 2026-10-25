<script lang="ts">
	import type { Task } from '$lib/types/task';
	import { tagFilterStore } from '$lib/stores/tagFilter.svelte';

	interface Props {
		// Unfiltered tasks of the current view; the picker offers their tags.
		tasks: Task[];
	}

	let { tasks }: Props = $props();
	let open = $state(false);
	let container: HTMLDivElement | undefined = $state();

	let tagCounts = $derived.by(() => {
		const counts = new Map<string, number>();
		for (const t of tasks) {
			for (const tag of t.tags ?? []) counts.set(tag, (counts.get(tag) ?? 0) + 1);
		}
		return [...counts].sort((a, b) => a[0].localeCompare(b[0]));
	});

	function select(tag: string) {
		tagFilterStore.tag = tag;
		open = false;
	}

	function handleWindowClick(e: MouseEvent) {
		if (open && container && !container.contains(e.target as Node)) open = false;
	}

	function handleWindowKeydown(e: KeyboardEvent) {
		if (open && e.key === 'Escape') open = false;
	}
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleWindowKeydown} />

{#if tagFilterStore.tag || tagCounts.length > 0}
	<div bind:this={container} class="relative inline-flex items-center">
		<div
			class="inline-flex items-center rounded-full border text-xs {tagFilterStore.tag
				? 'border-orange-300 bg-orange-200 text-orange-900 dark:border-yellow-400 dark:bg-yellow-500 dark:text-gray-900'
				: 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300'}"
		>
			<button
				onclick={() => (open = !open)}
				class="inline-flex items-center gap-1 pl-2.5 pr-1.5 py-1 rounded-full touch-manipulation"
				aria-haspopup="listbox"
				aria-expanded={open}
				aria-label="Filter by tag"
			>
				<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
				</svg>
				<span>{tagFilterStore.tag ?? 'Tags'}</span>
				<svg class="w-3.5 h-3.5 transition-transform" class:rotate-180={open} fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if tagFilterStore.tag}
				<button
					onclick={() => tagFilterStore.clear()}
					class="p-1 mr-0.5 rounded-full hover:bg-orange-300 dark:hover:bg-yellow-400 touch-manipulation"
					aria-label="Clear tag filter"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			{/if}
		</div>

		{#if open}
			<ul
				role="listbox"
				aria-label="Tags"
				class="absolute right-0 top-full mt-1 z-20 min-w-[10rem] max-h-64 overflow-y-auto py-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg"
			>
				{#each tagCounts as [tag, count] (tag)}
					<li role="option" aria-selected={tag === tagFilterStore.tag}>
						<button
							onclick={() => select(tag)}
							class="w-full flex items-center gap-3 px-3 py-1.5 text-left text-sm text-gray-800 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
							class:font-semibold={tag === tagFilterStore.tag}
						>
							<span class="flex-1 truncate">{tag}</span>
							<span class="text-xs text-gray-400 dark:text-gray-500">{count}</span>
						</button>
					</li>
				{:else}
					<li class="px-3 py-1.5 text-sm text-gray-500 dark:text-gray-400 italic">No tags in this view</li>
				{/each}
			</ul>
		{/if}
	</div>
{/if}
