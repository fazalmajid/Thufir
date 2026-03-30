<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import SidebarDropZone from './SidebarDropZone.svelte';
	import type { Task } from '$lib/types/task';

	interface Props {
		isOpen?: boolean;
		onClose?: () => void;
	}

	interface AreaWithProjects {
		area: typeof areaStore.areas[0] | null;
		projects: typeof projectStore.projects;
	}

	let { isOpen = true, onClose }: Props = $props();

	// Reactive computed values
	let inboxCount = $derived(taskStore.inboxTasks.length);
	let todayCount = $derived(taskStore.todayTasks.length);
	let completedCount = $derived(taskStore.completedTasks.length);
	let trashedCount = $derived(taskStore.trashedTasks.length);

	// Group projects by area, with active task counts
	let areasWithProjects = $derived.by(() => {
		const areas = areaStore.activeAreas;
		const tasks = taskStore.tasks;
		const result: (AreaWithProjects & { taskCount: number })[] = [];

		for (const area of areas) {
			const projects = projectStore.projectsByArea(area.id);
			const projectIds = new Set(projects.map(p => p.id));
			const taskCount = tasks.filter(t =>
				!t.is_completed && !t.deleted_at &&
				(t.area_id === area.id || (t.project_id != null && projectIds.has(t.project_id)))
			).length;
			result.push({ area, projects, taskCount });
		}

		// Projects without area
		const projectsWithoutArea = projectStore.projectsByArea(null);
		if (projectsWithoutArea.length > 0) {
			const projectIds = new Set(projectsWithoutArea.map(p => p.id));
			const taskCount = tasks.filter(t =>
				!t.is_completed && !t.deleted_at && t.project_id != null && projectIds.has(t.project_id)
			).length;
			result.push({ area: null, projects: projectsWithoutArea, taskCount });
		}

		return result;
	});

	let expandedAreas = $state<Set<string>>(new Set());

	onMount(() => {
		const match = document.cookie.match(/(?:^|; )expandedAreas=([^;]*)/);
		if (match) {
			try {
				expandedAreas = new Set(JSON.parse(decodeURIComponent(match[1])));
			} catch {}
		}
	});

	$effect(() => {
		document.cookie = `expandedAreas=${encodeURIComponent(JSON.stringify([...expandedAreas]))}; path=/`;
	});

	function toggleArea(areaId: string) {
		if (expandedAreas.has(areaId)) {
			expandedAreas.delete(areaId);
		} else {
			expandedAreas.add(areaId);
		}
		expandedAreas = new Set(expandedAreas); // Trigger reactivity
	}

	// Helper to check if route is active
	function isActive(path: string) {
		return $page.url.pathname === path;
	}

	// Close menu when clicking a link on mobile
	function handleLinkClick() {
		if (onClose) {
			onClose();
		}
	}

	function topSortOrder(targetTasks: { sort_order: number }[]): number {
		if (targetTasks.length === 0) return 0;
		return Math.min(...targetTasks.map((t) => t.sort_order)) - 1;
	}

	function dropToStatus(task: Task, status: string, extra: Record<string, unknown> = {}) {
		const target = taskStore.tasks.filter(
			(t) => t.status === status && !t.is_completed && !t.deleted_at
		);
		taskStore.update(task.id, { status, sort_order: topSortOrder(target), ...extra });
	}

	function dropToArea(task: Task, areaId: string) {
		const target = taskStore.tasks.filter(
			(t) => t.area_id === areaId && !t.project_id && !t.is_completed && !t.deleted_at
		);
		taskStore.update(task.id, { area_id: areaId, project_id: null, sort_order: topSortOrder(target) });
	}

	function dropToProject(task: Task, projectId: string, areaId: string | null) {
		const target = taskStore.tasks.filter(
			(t) => t.project_id === projectId && !t.is_completed && !t.deleted_at
		);
		taskStore.update(task.id, { project_id: projectId, area_id: areaId, sort_order: topSortOrder(target) });
	}

	// Search functionality
	let searchQuery = $state('');

	function handleSearch(e: Event) {
		e.preventDefault();
		if (searchQuery.trim()) {
			// Navigate to search page with query
			window.location.href = `/search?q=${encodeURIComponent(searchQuery.trim())}`;
		}
	}

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

	async function handleLogout() {
		await fetch(`${API_URL}/api/auth/logout`, { method: 'POST', credentials: 'include' });
		window.location.href = '/login';
	}
</script>

<!-- Mobile overlay -->
{#if isOpen}
	<div
		class="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
		onclick={onClose}
		onkeydown={(e) => e.key === 'Escape' && onClose?.()}
		role="button"
		tabindex="-1"
		aria-label="Close menu"
	></div>
{/if}

<!-- Sidebar -->
<aside
	class="w-64 bg-white border-r border-gray-200 flex flex-col h-screen
		fixed md:static inset-y-0 left-0 z-50
		transform transition-transform duration-300 ease-in-out
		{isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
	<div class="p-4 border-b border-gray-200">
		<h1 class="text-xl font-bold text-gray-900">Thufir</h1>
		<p class="text-xs text-gray-500 mt-1">Local-first tasks</p>
	</div>

	<!-- Search box -->
	<div class="px-4 pt-4 pb-2">
		<form onsubmit={handleSearch} class="relative">
			<div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
			</div>
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search tasks..."
				class="w-full pl-10 pr-3 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
		</form>
	</div>

	<nav class="flex-1 overflow-y-auto p-4 space-y-6">
		<!-- Main Views -->
		<div class="space-y-1">
			<SidebarDropZone onDrop={(task: Task) => dropToStatus(task, 'inbox', { area_id: null, project_id: null })}>
				<a
					href="/inbox"
					onclick={handleLinkClick}
					class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
					class:bg-blue-50={isActive('/inbox')}
					class:text-blue-700={isActive('/inbox')}
					class:font-medium={isActive('/inbox')}
				>
					<div class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
						</svg>
						<span>Inbox</span>
					</div>
					{#if inboxCount > 0}
						<span class="text-xs font-semibold text-gray-500">{inboxCount}</span>
					{/if}
				</a>
			</SidebarDropZone>

			<SidebarDropZone onDrop={(task: Task) => dropToStatus(task, 'today')}>
				<a
					href="/today"
					onclick={handleLinkClick}
					class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
					class:bg-blue-50={isActive('/today')}
					class:text-blue-700={isActive('/today')}
					class:font-medium={isActive('/today')}
				>
					<div class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
						<span>Today</span>
					</div>
					{#if todayCount > 0}
						<span class="text-xs font-semibold text-gray-500">{todayCount}</span>
					{/if}
				</a>
			</SidebarDropZone>

			<SidebarDropZone onDrop={(task: Task) => dropToStatus(task, 'upcoming')}>
				<a
					href="/upcoming"
					onclick={handleLinkClick}
					class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
					class:bg-blue-50={isActive('/upcoming')}
					class:text-blue-700={isActive('/upcoming')}
					class:font-medium={isActive('/upcoming')}
				>
					<div class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
						</svg>
						<span>Upcoming</span>
					</div>
				</a>
			</SidebarDropZone>

			<SidebarDropZone onDrop={(task: Task) => dropToStatus(task, 'anytime')}>
				<a
					href="/anytime"
					onclick={handleLinkClick}
					class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
					class:bg-blue-50={isActive('/anytime')}
					class:text-blue-700={isActive('/anytime')}
					class:font-medium={isActive('/anytime')}
				>
					<div class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
						</svg>
						<span>Anytime</span>
					</div>
				</a>
			</SidebarDropZone>

			<SidebarDropZone onDrop={(task: Task) => dropToStatus(task, 'someday')}>
				<a
					href="/someday"
					onclick={handleLinkClick}
					class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
					class:bg-blue-50={isActive('/someday')}
					class:text-blue-700={isActive('/someday')}
					class:font-medium={isActive('/someday')}
				>
					<div class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9" />
						</svg>
						<span>Someday</span>
					</div>
				</a>
			</SidebarDropZone>
		</div>

		<!-- Areas & Projects -->
		{#if areasWithProjects.length > 0}
			<div class="space-y-1">
				<h3 class="px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
					Areas & Projects
				</h3>

				{#each areasWithProjects as { area, projects, taskCount }}
					<div>
						{#if area}
							<!-- Area with toggle -->
							<SidebarDropZone onDrop={(task) => dropToArea(task, area.id)}>
								<a
									href="/areas/{area.id}"
									onclick={handleLinkClick}
									class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
									class:bg-blue-50={isActive(`/areas/${area.id}`)}
									class:text-blue-700={isActive(`/areas/${area.id}`)}
								>
									<div class="flex items-center gap-2 flex-1 min-w-0">
										<button
											onclick={(e) => { e.preventDefault(); e.stopPropagation(); toggleArea(area.id); }}
											class="flex-shrink-0 p-0.5 -ml-0.5 rounded hover:bg-gray-200 transition-colors"
											aria-label={expandedAreas.has(area.id) ? 'Collapse area' : 'Expand area'}
										>
											<svg
												class="w-4 h-4 transition-transform"
												class:rotate-90={expandedAreas.has(area.id)}
												fill="none"
												stroke="currentColor"
												viewBox="0 0 24 24"
											>
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
											</svg>
										</button>
										<span class="font-medium truncate">{area.name}</span>
									</div>
									{#if taskCount > 0}<span class="text-xs text-gray-500 flex-shrink-0">{taskCount}</span>{/if}
								</a>
							</SidebarDropZone>

							<!-- Projects under this area -->
							{#if expandedAreas.has(area.id)}
								<div class="ml-6 mt-1 space-y-1">
									{#each projects as project}
										<SidebarDropZone onDrop={(task) => dropToProject(task, project.id, project.area_id ?? null)}>
											<a
												onclick={handleLinkClick}
												href="/projects/{project.id}"
												class="flex items-center justify-between px-3 py-1.5 rounded-lg hover:bg-gray-100 transition-colors text-sm"
												class:bg-blue-50={isActive(`/projects/${project.id}`)}
												class:text-blue-700={isActive(`/projects/${project.id}`)}
											>
												<span>{project.name}</span>
											</a>
										</SidebarDropZone>
									{/each}
								</div>
							{/if}
						{:else}
							<!-- Projects without area -->
							<div class="space-y-1">
								<div class="px-3 py-1 text-sm text-gray-500 font-medium">Projects</div>
								{#each projects as project}
									<SidebarDropZone onDrop={(task) => dropToProject(task, project.id, null)}>
										<a
											onclick={handleLinkClick}
											href="/projects/{project.id}"
											class="flex items-center justify-between px-3 py-1.5 rounded-lg hover:bg-gray-100 transition-colors text-sm"
											class:bg-blue-50={isActive(`/projects/${project.id}`)}
											class:text-blue-700={isActive(`/projects/${project.id}`)}
										>
											<span>{project.name}</span>
										</a>
									</SidebarDropZone>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<!-- Logbook (Completed) & Trash -->
		<div class="space-y-1 pt-4 border-t border-gray-200">
			<a
				href="/logbook"
				onclick={handleLinkClick}
				class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
				class:bg-blue-50={isActive('/logbook')}
				class:text-blue-700={isActive('/logbook')}
				class:font-medium={isActive('/logbook')}
			>
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span>Logbook</span>
				</div>
				{#if completedCount > 0}
					<span class="text-xs font-semibold text-gray-500">{completedCount}</span>
				{/if}
			</a>

			<a
				href="/trash"
				onclick={handleLinkClick}
				class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-gray-100 transition-colors"
				class:bg-blue-50={isActive('/trash')}
				class:text-blue-700={isActive('/trash')}
				class:font-medium={isActive('/trash')}
			>
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
					</svg>
					<span>Trash</span>
				</div>
				{#if trashedCount > 0}
					<span class="text-xs font-semibold text-gray-500">{trashedCount}</span>
				{/if}
			</a>
		</div>
	</nav>

	<div class="p-4 border-t border-gray-200 space-y-1">
		<a
			href="/settings"
			onclick={handleLinkClick}
			class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-500 rounded-lg hover:bg-gray-100 transition-colors"
			class:bg-blue-50={isActive('/settings')}
			class:text-blue-700={isActive('/settings')}
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
			</svg>
			Settings
		</a>
		<button
			onclick={handleLogout}
			class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-500 rounded-lg hover:bg-gray-100 transition-colors"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
			</svg>
			Sign out
		</button>
	</div>
</aside>
