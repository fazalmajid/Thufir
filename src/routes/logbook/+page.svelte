<script lang="ts">
	import { onMount } from 'svelte';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';

	onMount(() => {
		taskStore.load({ status: 'completed' });
	});
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	<div class="mb-4">
		<h1 class="text-2xl font-bold text-gray-900">Logbook</h1>
		<p class="text-sm text-gray-600 mt-0.5">Completed tasks</p>
	</div>

	{#if taskStore.completedTasks.length === 0}
		<div class="bg-white rounded-lg shadow-sm p-4 text-center">
			<svg class="w-10 h-10 mx-auto text-gray-300 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<p class="text-sm text-gray-500">No completed tasks yet</p>
			<p class="text-xs text-gray-400 mt-0.5">Complete tasks to see them here</p>
		</div>
	{:else}
		<div class="bg-white rounded-lg shadow-sm p-0">
			<TaskList tasks={taskStore.completedTasks} title="" />
		</div>
	{/if}
</div>
