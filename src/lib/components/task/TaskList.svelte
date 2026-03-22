<script lang="ts">
	import type { Task } from '$lib/types/task';
	import TaskItem from './TaskItem.svelte';
	import { dndzone, TRIGGERS } from 'svelte-dnd-action';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { dragStore } from '$lib/stores/drag.svelte';
	import { flip } from 'svelte/animate';

	interface Props {
		tasks: Task[];
		title: string;
		enableReorder?: boolean;
		showContext?: boolean;
	}

	let { tasks, title, enableReorder = false, showContext = false }: Props = $props();
	let items = $state.raw<Task[]>([...tasks]);
	let isDragging = $state(false);
	// When a task is dropped on a sidebar zone, exclude it from the synced list
	// until the API confirms the move (at which point it's gone from tasks anyway).
	let excludeId = $state<string | null>(null);

	$effect(() => {
		if (isDragging) return;
		// Once the task is actually gone from tasks, clear the exclusion.
		if (excludeId && !tasks.some(t => t.id === excludeId)) excludeId = null;
		items = excludeId ? tasks.filter(t => t.id !== excludeId) : [...tasks];
	});

	function handleDndConsider(e: CustomEvent<{ items: Task[]; info: { trigger: string; id: string } }>) {
		const { items: newItems, info } = e.detail;
		if (info.trigger === TRIGGERS.DRAG_STARTED) {
			isDragging = true;
			dragStore.task = tasks.find(t => t.id === info.id) ?? newItems.find((t: Task) => t.id === info.id) ?? null;
		}
		items = newItems;
	}

	async function handleDndFinalize(e: CustomEvent<{ items: Task[]; info: { trigger: string; id: string } }>) {
		const { items: finalItems, info } = e.detail;
		const droppedToZone = dragStore.dropped; // capture before clear()
		isDragging = false;
		dragStore.clear();

		if (info.trigger === TRIGGERS.DROPPED_OUTSIDE_OF_ANY) {
			if (droppedToZone) {
				// The $effect will filter this task out until the API confirms removal.
				excludeId = info.id;
			}
			return;
		}

		items = finalItems;

		if (enableReorder) {
			try {
				await taskStore.reorder(items);
			} catch (err) {
				console.error('Failed to reorder tasks:', err);
				items = [...tasks];
			}
		}
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
				dragDisabled: !enableReorder,
				dropTargetStyle: {},
				type: 'tasks'
			}}
			onconsider={handleDndConsider}
			onfinalize={handleDndFinalize}
		>
			{#each items as task (task.id)}
				<div animate:flip={{ duration: 200 }}>
					<TaskItem {task} draggable={enableReorder} {showContext} />
				</div>
			{/each}
		</div>
	{/if}
</div>
