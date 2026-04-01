<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	let error = $state('');

	onMount(async () => {
		const title = $page.url.searchParams.get('title') ?? '';
		const url = $page.url.searchParams.get('url') ?? '';

		try {
			const res = await fetch('/api/tasks/quick-add', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ title, notes: url || null })
			});

			if (res.ok) {
				goto('/inbox');
			} else {
				const body = await res.json().catch(() => ({}));
				error = body.error ?? `HTTP ${res.status}`;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Network error';
		}
	});
</script>

<svelte:head>
	<title>Add to Thufir</title>
</svelte:head>

{#if error}
	<div class="min-h-screen bg-white flex items-center justify-center p-6">
		<div class="text-center space-y-3">
			<p class="text-sm font-medium text-gray-800">Failed to save</p>
			<p class="text-xs text-red-500">{error}</p>
		</div>
	</div>
{/if}
