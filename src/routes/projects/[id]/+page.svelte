<script lang="ts">
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';

	let projectId = $derived($page.params.id);
	let project = $derived(projectStore.projects.find(p => p.id === projectId));
	let projectTasks = $derived(taskStore.tasks.filter(
		t => t.project_id === projectId && !t.is_completed && !t.deleted_at
	));
</script>

<div class="container mx-auto p-8 max-w-4xl">
	{#if project}
		<div class="mb-6">
			<h1 class="text-3xl font-bold text-gray-900">{project.name}</h1>
			{#if project.notes}
				<p class="text-gray-600 mt-2">{project.notes}</p>
			{/if}
		</div>

		<div class="bg-white rounded-lg shadow p-6 mb-6">
			<TaskQuickAdd status="inbox" />
		</div>

		<TaskList tasks={projectTasks} title="" />
	{:else}
		<div class="bg-white rounded-lg shadow p-12 text-center">
			<p class="text-gray-500">Project not found</p>
		</div>
	{/if}
</div>
