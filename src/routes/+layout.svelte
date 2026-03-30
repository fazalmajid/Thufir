<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import { dragStore } from '$lib/stores/drag.svelte';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';
	import '../app.css';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

	let { children }: { children: Snippet } = $props();
	let isMobileMenuOpen = $state(false);
	let authReady = $state(false);
	let isAuthenticated = $state(false);

	function toggleMobileMenu() {
		isMobileMenuOpen = !isMobileMenuOpen;
	}

	function closeMobileMenu() {
		isMobileMenuOpen = false;
	}

	$effect(() => {
		if (!dragStore.task) return;

		function onMove(e: PointerEvent) {
			dragStore.updateActiveZone(e.clientX, e.clientY);
		}
		function onUp() {
			dragStore.drop();
		}

		document.addEventListener('pointermove', onMove, { passive: true });
		document.addEventListener('pointerup', onUp, { capture: true, once: true });

		return () => {
			document.removeEventListener('pointermove', onMove);
			document.removeEventListener('pointerup', onUp, { capture: true });
		};
	});

	onMount(async () => {
		if ($page.url.pathname === '/login') {
			authReady = true;
			return;
		}

		try {
			const res = await fetch(`${API_URL}/api/auth/me`, { credentials: 'include' });
			if (!res.ok) {
				goto('/login');
				return;
			}
		} catch {
			goto('/login');
			return;
		}

		isAuthenticated = true;
		authReady = true;

		const activeStatuses = ['inbox', 'today', 'upcoming', 'anytime', 'someday'];
		Promise.all(activeStatuses.map((status) => taskStore.load({ status })));
		projectStore.load();
		areaStore.load();
	});
</script>

{#if $page.url.pathname === '/login'}
	{@render children()}
{:else if authReady && isAuthenticated}
	<div class="flex h-screen overflow-hidden">
		<Sidebar isOpen={isMobileMenuOpen} onClose={closeMobileMenu} />
		<div class="flex-1 flex flex-col overflow-hidden">
			<Header onMenuToggle={toggleMobileMenu} />
			<main class="flex-1 overflow-y-auto bg-gray-50">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
