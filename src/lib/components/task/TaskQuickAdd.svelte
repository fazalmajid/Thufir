<script lang="ts">
	import { taskStore } from '$lib/stores/tasks.svelte';
	import type { TaskStatus } from '$lib/types/task';

	interface Props {
		status?: TaskStatus;
	}

	let { status = 'inbox' }: Props = $props();

	let title = $state('');
	let isSubmitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!title.trim() || isSubmitting) return;

		isSubmitting = true;

		try {
			await taskStore.create({
				title: title.trim(),
				status
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
		class="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
	/>
	<button
		type="submit"
		disabled={!title.trim() || isSubmitting}
		class="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed"
	>
		{isSubmitting ? 'Adding...' : 'Add'}
	</button>
</form>
