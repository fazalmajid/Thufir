<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import type { TaskStatus } from '$lib/types/task';

	interface Props {
		status?: TaskStatus;
		area_id?: string;
		project_id?: string;
	}

	let { status = 'inbox', area_id, project_id }: Props = $props();

	let title = $state('');
	let isSubmitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!title.trim() || isSubmitting) return;

		isSubmitting = true;

		try {
			const contextTasks = taskStore.tasks.filter(t => {
				if (!t.is_completed && !t.deleted_at) {
					if (project_id) return t.project_id === project_id;
					if (area_id) return t.area_id === area_id && !t.project_id;
					return t.status === status;
				}
				return false;
			});
			const sort_order = contextTasks.length > 0
				? Math.min(...contextTasks.map(t => t.sort_order)) - 1
				: 0;

			await taskStore.create({
				title: title.trim(),
				status,
				sort_order,
				...(area_id ? { area_id } : {}),
				...(project_id ? { project_id } : {})
			});
			title = '';
		} catch (err) {
			console.error('Failed to create task:', err);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} class="flex gap-2">
	<input
		type="text"
		bind:value={title}
		placeholder="Add a new task..."
		disabled={isSubmitting}
		class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
	/>
	<button
		type="submit"
		disabled={!title.trim() || isSubmitting}
		class="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-gray-600 disabled:cursor-not-allowed"
	>
		{isSubmitting ? 'Adding...' : 'Add'}
	</button>
</form>
