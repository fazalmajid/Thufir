<script lang="ts">
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import type { Task } from '$lib/types/task';

	// Get search query from URL
	let searchQuery = $derived($page.url.searchParams.get('q') || '');

	// Filter tasks based on search query
	let searchResults = $derived.by(() => {
		if (!searchQuery.trim()) {
			return [];
		}

		const query = searchQuery.toLowerCase().trim();
		const allTasks = taskStore.tasks;

		const matches = allTasks.filter((task: Task) => {
			// Search in title
			if (task.title.toLowerCase().includes(query)) {
				return true;
			}

			// Search in notes
			if (task.notes && task.notes.toLowerCase().includes(query)) {
				return true;
			}

			// Search in tags
			if (task.tags && task.tags.some(tag => tag.toLowerCase().includes(query))) {
				return true;
			}

			return false;
		});
		matches.sort((a: Task, b: Task) => Number(a.completed) - Number(b.completed));
		return matches;
	});

	let resultCount = $derived(searchResults.length);
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	<div class="mb-4">
		<h1 class="text-2xl font-bold text-gray-900">Search Results</h1>
		{#if searchQuery}
			<p class="text-sm text-gray-600 mt-0.5">
				Found {resultCount} {resultCount === 1 ? 'task' : 'tasks'} matching "{searchQuery}"
			</p>
		{:else}
			<p class="text-sm text-gray-600 mt-0.5">Enter a search query to find tasks</p>
		{/if}
	</div>

	{#if searchQuery}
		{#if resultCount > 0}
			<div class="bg-white rounded-lg shadow-sm p-0">
				<TaskList tasks={searchResults} title="" enableReorder={false} showContext={true} />
			</div>
		{:else}
			<div class="bg-white rounded-lg shadow-sm p-8 text-center">
				<svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<h3 class="text-lg font-medium text-gray-900 mb-1">No results found</h3>
				<p class="text-sm text-gray-500">Try searching with different keywords</p>
			</div>
		{/if}
	{/if}
</div>
