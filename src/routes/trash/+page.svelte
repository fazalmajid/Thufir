<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import type { Task } from '$lib/types/task';
	import TagFilter from '$lib/components/task/TagFilter.svelte';
	import { tagFilterStore } from '$lib/stores/tagFilter.svelte';

	async function handleRestore(task: Task) {
		try {
			await taskStore.restore(task.id);
		} catch (err) {
			console.error('Failed to restore task:', err);
		}
	}

	async function handlePermanentDelete(task: Task) {
		if (!confirm('Permanently delete this task? This cannot be undone.')) {
			return;
		}

		try {
			// For now, this just removes from client state
			// In production, you'd call a different API endpoint for hard delete
			taskStore.tasks = taskStore.tasks.filter(t => t.id !== task.id);
		} catch (err) {
			console.error('Failed to permanently delete task:', err);
		}
	}

	async function handleEmptyTrash() {
		if (!confirm('Permanently delete all items in trash? This cannot be undone.')) {
			return;
		}

		try {
			const trashedIds = taskStore.trashedTasks.map(t => t.id);
			taskStore.tasks = taskStore.tasks.filter(t => !trashedIds.includes(t.id));
		} catch (err) {
			console.error('Failed to empty trash:', err);
		}
	}
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	<div class="mb-4 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Trash</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400 mt-0.5">Deleted tasks</p>
		</div>
		<div class="flex items-center gap-2">
			<TagFilter tasks={taskStore.trashedTasks} />
			{#if taskStore.trashedTasks.length > 0}
				<button
					onclick={handleEmptyTrash}
					class="px-3 py-1.5 text-xs text-red-600 hover:text-red-700 border border-red-300 rounded-lg hover:bg-red-50 transition-colors"
				>
					Empty Trash
				</button>
			{/if}
		</div>
	</div>

	{#if taskStore.trashedTasks.length === 0}
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4 text-center">
			<svg class="w-10 h-10 mx-auto text-gray-300 dark:text-gray-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
			</svg>
			<p class="text-sm text-gray-500 dark:text-gray-400">Trash is empty</p>
			<p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">Deleted tasks will appear here</p>
		</div>
	{:else}
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm">
			{#each tagFilterStore.apply(taskStore.trashedTasks) as task (task.id)}
				<div class="flex items-center gap-3 py-2 px-1 hover:bg-gray-50 dark:hover:bg-gray-700 border-b border-gray-100 dark:border-gray-700 last:border-0">
					<div class="flex-1 min-w-0">
						<p class="text-sm text-gray-500 dark:text-gray-400 line-through">
							{task.title}
						</p>
						{#if task.notes}
							<p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5 line-through">{task.notes}</p>
						{/if}
						{#if task.tags && task.tags.length > 0}
							<div class="flex flex-wrap gap-1.5 mt-1">
								{#each task.tags as tag}
									<button
										onclick={() => tagFilterStore.toggle(tag)}
										class="text-xs px-1.5 py-0.5 rounded {tag === tagFilterStore.tag
											? 'bg-blue-600 text-white hover:bg-blue-700'
											: 'bg-blue-100 text-blue-700 hover:bg-blue-200'}"
										aria-pressed={tag === tagFilterStore.tag}
									>{tag}</button>
								{/each}
							</div>
						{/if}
						{#if task.deleted_at}
							<p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">
								Deleted {new Date(task.deleted_at).toLocaleDateString()}
							</p>
						{/if}
					</div>

					<div class="flex gap-2">
						<button
							onclick={() => handleRestore(task)}
							class="text-blue-600 hover:text-blue-700 text-xs px-2 py-1 border border-blue-300 rounded hover:bg-blue-50 transition-colors"
						>
							Restore
						</button>
						<button
							onclick={() => handlePermanentDelete(task)}
							class="text-red-600 hover:text-red-700 text-xs px-2 py-1 border border-red-300 rounded hover:bg-red-50 transition-colors"
						>
							Delete Forever
						</button>
					</div>
				</div>
			{/each}
		</div>

		<div class="mt-2 p-2 bg-yellow-50 border border-yellow-200 rounded-lg">
			<p class="text-xs text-yellow-800">
				<strong>Note:</strong> Tasks in trash are currently stored with soft delete.
				In production, you may want to automatically purge items after 30 days.
			</p>
		</div>
	{/if}
</div>
