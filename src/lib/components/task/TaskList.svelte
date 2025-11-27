<script lang="ts">
	import type { Task } from '$lib/types/task';
	import TaskItem from './TaskItem.svelte';

	interface Props {
		tasks: Task[];
		title: string;
	}

	let { tasks, title }: Props = $props();
</script>

<div>
	{#if title}
		<h2 class="text-xl font-semibold text-gray-900 mb-2">{title}</h2>
	{/if}

	{#if tasks.length === 0}
		<p class="text-sm text-gray-500 italic py-4">No tasks</p>
	{:else}
		<div class="space-y-0.5">
			{#each tasks as task (task.id)}
				<TaskItem {task} />
			{/each}
		</div>
	{/if}
</div>
