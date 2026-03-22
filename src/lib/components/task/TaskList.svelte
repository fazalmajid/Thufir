<script lang="ts">
	import type { Task } from '$lib/types/task';
	import TaskItem from './TaskItem.svelte';
	import { dndzone } from 'svelte-dnd-action';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { flip } from 'svelte/animate';
	import { untrack } from 'svelte';

	interface Props {
		tasks: Task[];
		title: string;
		enableReorder?: boolean;
		showContext?: boolean;
	}

	let { tasks, title, enableReorder = false, showContext = false }: Props = $props();
	let items = $state.raw<Task[]>([...tasks]);

	// Only sync items when the SET of tasks changes (add/remove),
	// not when only order changes (DnD manages order locally)
	$effect(() => {
		const newTaskIds = new Set(tasks.map(t => t.id));
		const changed = untrack(() => {
			if (newTaskIds.size !== items.length) return true;
			return items.some(t => !newTaskIds.has(t.id));
		});
		if (changed) {
			items = [...tasks];
		}
	});

	function handleDndConsider(e: CustomEvent<{ items: Task[] }>) {
		items = e.detail.items;
	}

	async function handleDndFinalize(e: CustomEvent<{ items: Task[] }>) {
		items = e.detail.items;

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
