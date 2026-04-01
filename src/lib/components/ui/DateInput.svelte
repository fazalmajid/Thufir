<script lang="ts">
	interface Props {
		value: string;
		placeholder?: string;
		class?: string;
	}

	let { value = $bindable(), placeholder = 'YYYY-MM-DD', class: cls = '' }: Props = $props();

	let picker: HTMLInputElement | undefined = $state();

	function openPicker() {
		picker?.showPicker();
	}
</script>

<div class="relative flex items-center">
	<input
		type="text"
		bind:value
		{placeholder}
		pattern="\d{4}-\d{2}-\d{2}"
		class="w-full pr-7 {cls}"
	/>
	<button
		type="button"
		onclick={openPicker}
		tabindex="-1"
		class="absolute right-1.5 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-400"
		aria-label="Open date picker"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
				d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
		</svg>
	</button>
	<input
		bind:this={picker}
		type="date"
		bind:value
		tabindex="-1"
		class="sr-only"
		aria-hidden="true"
	/>
</div>
