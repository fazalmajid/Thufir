<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import { dragStore } from '$lib/stores/drag.svelte';
	import { startReplication } from '$lib/db/replication';
	import { getDB } from '$lib/db/index';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';
	import '../app.css';

	let { children }: { children: Snippet } = $props();
	let isMobileMenuOpen = $state(false);
	let authReady = $state(false);
	let isAuthenticated = $state(false);
	let initError = $state<string | null>(null);
	let dbInitialized = $state(false);

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

	$effect(() => {
		if ($page.url.pathname === '/login') {
			authReady = true;
			return;
		}
		if (isAuthenticated) return;

		// Check authentication whenever we navigate to a non-login page.
		(async () => {
			try {
				const res = await fetch('/api/auth/me', { credentials: 'include' });
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

			if (!dbInitialized) {
				dbInitialized = true;
				try {
					const db = await getDB();
					await Promise.all([taskStore.init(), areaStore.init(), projectStore.init()]);
					await startReplication(db);
				} catch (e) {
					initError = e instanceof Error ? e.message : String(e);
				}
			}
		})();
	});
</script>

{#if $page.url.pathname === '/login'}
	{@render children()}
{:else if authReady && isAuthenticated}
	<div class="flex h-screen overflow-hidden">
		<Sidebar isOpen={isMobileMenuOpen} onClose={closeMobileMenu} />
		<div class="flex-1 flex flex-col overflow-hidden">
			<Header onMenuToggle={toggleMobileMenu} />
			{#if initError}
				<div class="bg-red-50 border-b border-red-200 px-4 py-2 text-sm text-red-700">
					DB init failed: {initError}
				</div>
			{/if}
			<main class="flex-1 overflow-y-auto bg-gray-50">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
