<script lang="ts">
	import type { Task } from '$lib/types/task';
	import { taskStore } from '$lib/stores/tasks.svelte';

	interface Props {
		task: Task;
	}

	let { task }: Props = $props();

	async function handleToggle() {
		await taskStore.toggleComplete(task.id);
	}

	async function handleDelete() {
		if (confirm('Delete this task?')) {
			await taskStore.delete(task.id);
		}
	}
</script>

<div class="flex items-center gap-3 py-2 px-1 hover:bg-gray-50 rounded group">
	<input
		type="checkbox"
		checked={task.is_completed}
		onchange={handleToggle}
		class="w-4 h-4 rounded border-gray-300"
	/>

	<div class="flex-1 min-w-0">
		<p class="text-sm text-gray-900" class:line-through={task.is_completed} class:text-gray-500={task.is_completed}>
			{task.title}
		</p>
		{#if task.notes}
			<p class="text-xs text-gray-500 mt-0.5">{task.notes}</p>
		{/if}
		{#if task.tags && task.tags.length > 0}
			<div class="flex gap-1.5 mt-1">
				{#each task.tags as tag}
					<span class="text-xs px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded">{tag}</span>
				{/each}
			</div>
		{/if}
	</div>

	<button
		onclick={handleDelete}
		class="text-gray-400 hover:text-red-600 text-xs px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity"
	>
		Delete
	</button>
</div>
