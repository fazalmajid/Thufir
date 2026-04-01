<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';

	const PAGE_SIZE = 100;
	let page = $state(0);

	let allTasks = $derived(taskStore.somedayTasks);
	let visibleTasks = $derived(allTasks.slice(0, (page + 1) * PAGE_SIZE));
	let hasMore = $derived(visibleTasks.length < allTasks.length);
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	<div class="mb-4">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Someday</h1>
		<p class="text-sm text-gray-600 dark:text-gray-400 mt-0.5">Ideas and tasks for the future</p>
	</div>

	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-3 mb-3">
		<TaskQuickAdd status="someday" />
	</div>

	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-0">
		<TaskList tasks={visibleTasks} title="" enableReorder={false} />
	</div>

	{#if hasMore}
		<button
			onclick={() => page++}
			class="mt-3 w-full py-2 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-white dark:hover:bg-gray-800 rounded-lg transition-colors"
		>
			Show more ({allTasks.length - visibleTasks.length} remaining)
		</button>
	{/if}
</div>
