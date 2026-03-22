<script lang="ts">
	import { onMount } from 'svelte';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';
	import '../app.css';

	let isMobileMenuOpen = $state(false);

	function toggleMobileMenu() {
		isMobileMenuOpen = !isMobileMenuOpen;
	}

	function closeMobileMenu() {
		isMobileMenuOpen = false;
	}

	onMount(() => {
		// Load only active tasks on startup (excludes 6000+ completed tasks)
		// Logbook loads completed tasks on demand; project pages load their own tasks
		const activeStatuses = ['inbox', 'today', 'upcoming', 'anytime', 'someday'];
		Promise.all(activeStatuses.map((status) => taskStore.load({ status })));
		projectStore.load();
		areaStore.load();
	});
</script>

<div class="flex h-screen overflow-hidden">
	<Sidebar isOpen={isMobileMenuOpen} onClose={closeMobileMenu} />
	<div class="flex-1 flex flex-col overflow-hidden">
		<Header onMenuToggle={toggleMobileMenu} />
		<main class="flex-1 overflow-y-auto bg-gray-50">
			<slot />
		</main>
	</div>
</div>
