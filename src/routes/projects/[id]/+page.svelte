<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';

	let projectId = $derived($page.params.id);
	let project = $derived(projectStore.projects.find(p => p.id === projectId));
	let projectTasks = $derived(
		taskStore.tasks
			.filter(t => t.project_id === projectId && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order)
	);

	onMount(() => {
		taskStore.load({ project_id: projectId });
	});
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	{#if project}
		<div class="mb-4">
			<h1 class="text-2xl font-bold text-gray-900">{project.name}</h1>
			{#if project.notes}
				<p class="text-sm text-gray-600 mt-0.5">{project.notes}</p>
			{/if}
		</div>

		<div class="bg-white rounded-lg shadow-sm p-3 mb-3">
			<TaskQuickAdd status="inbox" />
		</div>

		<div class="bg-white rounded-lg shadow-sm p-0">
			<TaskList tasks={projectTasks} title="" enableReorder={true} />
		</div>
	{:else}
		<div class="bg-white rounded-lg shadow-sm p-4 text-center">
			<p class="text-sm text-gray-500">Project not found</p>
		</div>
	{/if}
</div>
