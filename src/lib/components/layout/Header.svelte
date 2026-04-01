<script lang="ts">
	import { syncError, lastSync } from '$lib/stores/sync';

	interface Props {
		onMenuToggle: () => void;
	}

	let { onMenuToggle }: Props = $props();

	function fmtTime(d: Date) {
		return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<header class="md:hidden bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
	<button
		onclick={onMenuToggle}
		class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
		aria-label="Toggle menu"
	>
		<svg class="w-6 h-6 text-gray-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
		</svg>
	</button>

	<h1 class="text-lg font-bold text-gray-900">Thufir</h1>

	<div class="w-10 flex justify-end">
		{#if $syncError}
			<span class="text-xs text-red-600 font-medium truncate max-w-[6rem]" title={$syncError}>
				⚠ {$syncError}
			</span>
		{:else if $lastSync}
			<span class="text-xs text-gray-400">{fmtTime($lastSync)}</span>
		{/if}
	</div>
</header>
