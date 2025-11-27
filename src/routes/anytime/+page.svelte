<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';
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
		<TaskList tasks={taskStore.anytimeTasks} title="" />
	</div>
</div>
