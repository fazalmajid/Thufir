<script lang="ts">
	interface Props {
		onMenuToggle: () => void;
	}

	let { onMenuToggle }: Props = $props();
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

	<div class="w-10"></div> <!-- Spacer for centering -->
</header>
