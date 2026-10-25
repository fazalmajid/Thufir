<script lang="ts">
	import { page } from '$app/state';
	import { taskStore } from '#lib/stores/tasks.svelte.js';
	import { projectStore } from '#lib/stores/projects.svelte.js';
	import TaskList from '#lib/components/task/TaskList.svelte';
	import TagFilter from '#lib/components/task/TagFilter.svelte';
	import { tagFilterStore } from '#lib/stores/tagFilter.svelte.js';
	import TaskQuickAdd from '#lib/components/task/TaskQuickAdd.svelte';

	let projectId = $derived(page.params.id);
	let project = $derived(projectStore.projects.find((p) => p.id === projectId));
	let projectTasks = $derived(taskStore.tasks.filter((t) => t.project_id === projectId && !t.is_completed && !t.deleted_at).sort((a, b) => a.sort_order - b.sort_order));
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	{#if project}
		<div class="mb-4 flex items-start justify-between gap-3">
			<div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{project.name}</h1>
				{#if project.notes}
					<p class="text-sm text-gray-600 dark:text-gray-400 mt-0.5">{project.notes}</p>
				{/if}
			</div>
			<TagFilter tasks={projectTasks} />
		</div>

		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-3 mb-3">
			<TaskQuickAdd status="anytime" project_id={projectId} />
		</div>

		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-0">
			<TaskList tasks={tagFilterStore.apply(projectTasks)} title="" enableReorder={true} />
		</div>
	{:else}
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4 text-center">
			<p class="text-sm text-gray-500 dark:text-gray-400">Project not found</p>
		</div>
	{/if}
</div>
