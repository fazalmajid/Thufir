<script lang="ts">
	import type { Task } from '$lib/types/task';
	import TaskItem from './TaskItem.svelte';
	import { dndzone } from 'svelte-dnd-action';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { flip } from 'svelte/animate';

	interface Props {
		tasks: Task[];
		title: string;
		enableReorder?: boolean;
	}

	let { tasks, title, enableReorder = false }: Props = $props();
	let items = $state<Task[]>([...tasks]);
	let dragDisabled = $state(true);

	// Sync items when tasks prop changes
	$effect(() => {
		items = [...tasks];
	});

	function handleDndConsider(e: CustomEvent<{ items: Task[] }>) {
		items = e.detail.items;
	}

	async function handleDndFinalize(e: CustomEvent<{ items: Task[] }>) {
		items = e.detail.items;

		// Only save if order actually changed
		if (enableReorder) {
			try {
				await taskStore.reorder(items);
			} catch (err) {
				console.error('Failed to reorder tasks:', err);
			}
		}
	}

	function handleMouseDown() {
		if (enableReorder) {
			dragDisabled = false;
		}
	}

	function handleMouseUp() {
		dragDisabled = true;
	}
</script>

<div>
	{#if title}
		<h2 class="text-xl font-semibold text-gray-900 mb-2">{title}</h2>
	{/if}

	{#if items.length === 0}
		<p class="text-sm text-gray-500 italic py-4">No tasks</p>
	{:else}
		<div
			class="space-y-0.5"
			use:dndzone={{
				items,
				dragDisabled,
				dropTargetStyle: {},
				type: 'tasks'
			}}
			onconsider={handleDndConsider}
			onfinalize={handleDndFinalize}
			onmousedown={handleMouseDown}
			onmouseup={handleMouseUp}
			ontouchstart={handleMouseDown}
			ontouchend={handleMouseUp}
		>
			{#each items as task (task.id)}
				<div animate:flip={{ duration: 200 }}>
					<TaskItem {task} draggable={enableReorder} />
				</div>
			{/each}
		</div>
	{/if}
</div>
