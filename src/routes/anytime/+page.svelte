<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';

	const PAGE_SIZE = 100;
	let page = $state(0);

	let allTasks = $derived(taskStore.anytimeTasks);
	let visibleTasks = $derived(allTasks.slice(0, (page + 1) * PAGE_SIZE));
	let hasMore = $derived(visibleTasks.length < allTasks.length);
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	<div class="mb-4">
		<h1 class="text-2xl font-bold text-gray-900">Anytime</h1>
		<p class="text-sm text-gray-600 mt-0.5">Tasks to do when you have time</p>
	</div>

	<div class="bg-white rounded-lg shadow-sm p-3 mb-3">
		<TaskQuickAdd status="anytime" />
	</div>

	<div class="bg-white rounded-lg shadow-sm p-0">
		<TaskList tasks={visibleTasks} title="" enableReorder={false} />
	</div>

	{#if hasMore}
		<button
			onclick={() => page++}
			class="mt-3 w-full py-2 text-sm text-gray-500 hover:text-gray-700 hover:bg-white rounded-lg transition-colors"
		>
			Show more ({allTasks.length - visibleTasks.length} remaining)
		</button>
	{/if}
</div>
