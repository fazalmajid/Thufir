<script lang="ts">
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import TaskList from '$lib/components/task/TaskList.svelte';
	import TaskQuickAdd from '$lib/components/task/TaskQuickAdd.svelte';

	let areaId = $derived($page.params.id);
	let area = $derived(areaStore.areas.find(a => a.id === areaId));
	let areaProjects = $derived(projectStore.projects.filter(p => p.area_id === areaId && !p.deleted_at));

	// Tasks directly on the area (no project), not completed
	let areaTasks = $derived(
		taskStore.tasks
			.filter(t => t.area_id === areaId && !t.project_id && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order)
	);
</script>

<div class="container mx-auto px-4 py-4 max-w-4xl">
	{#if area}
		<div class="mb-4">
			<h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{area.name}</h1>
		</div>

		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-3 mb-3">
			<TaskQuickAdd status="anytime" />
		</div>

		{#if areaTasks.length > 0}
			<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-0 mb-4">
				<TaskList tasks={areaTasks} title="" enableReorder={true} />
			</div>
		{/if}

		{#if areaProjects.length > 0}
			<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm divide-y divide-gray-100 dark:divide-gray-700">
				{#each areaProjects as project}
					{@const count = taskStore.tasks.filter(
						t => t.project_id === project.id && !t.is_completed && !t.deleted_at
					).length}
					<a
						href="/projects/{project.id}"
						class="flex items-center px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
					>
						<span class="font-medium text-gray-900 dark:text-gray-100">{project.name}</span>
						{#if count > 0}
							<span class="text-xs text-gray-400 dark:text-gray-500 ml-auto">{count}</span>
						{/if}
					</a>
				{/each}
			</div>
		{/if}

		{#if areaTasks.length === 0 && areaProjects.length === 0}
			<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-8 text-center">
				<p class="text-sm text-gray-500 dark:text-gray-400">No tasks or projects in this area</p>
			</div>
		{/if}
	{:else}
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-4 text-center">
			<p class="text-sm text-gray-500 dark:text-gray-400">Area not found</p>
		</div>
	{/if}
</div>
