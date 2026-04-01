<script lang="ts">
	import type { Task } from '$lib/types/task';
	import { dragStore } from '$lib/stores/drag.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		onDrop: (task: Task) => void | Promise<void>;
		children: Snippet;
	}

	let { onDrop, children }: Props = $props();
	let zoneEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!zoneEl) return;
		return dragStore.registerZone(zoneEl, onDrop);
	});

	let isOver = $derived(dragStore.activeZone?.el === zoneEl && zoneEl !== null);
</script>

<div
	bind:this={zoneEl}
	class="rounded-lg transition-colors {isOver && dragStore.task ? 'ring-2 ring-blue-400 bg-blue-50 dark:bg-blue-900/30' : ''}"
>
	{@render children()}
</div>
