<script lang="ts">
	import { onMount } from 'svelte';
	import { startRegistration } from '@simplewebauthn/browser';

	const ORIGIN = 'https://thufir.majid.org';

	const bookmarkletCode = `(function(){
var t=encodeURIComponent(document.title);
var u=encodeURIComponent(location.href);
window.open('${ORIGIN}/quick-add?title='+t+'&url='+u,'_blank');
})();`;

	const bookmarkletHref = 'javascript:' + encodeURIComponent(bookmarkletCode);

	

	interface Device {
		id: string;
		device_name: string | null;
		transports: string[];
		created_at: string;
	}

	let devices = $state<Device[]>([]);
	let loading = $state(true);
	let enrolling = $state(false);
	let newDeviceName = $state('');
	let error = $state('');
	let success = $state('');

	async function loadDevices() {
		const res = await fetch(`/api/auth/devices`, { credentials: 'include' });
		devices = await res.json();
		loading = false;
	}

	async function enroll() {
		enrolling = true;
		error = '';
		success = '';
		try {
			const optRes = await fetch(`/api/auth/device/options`, {
				method: 'POST',
				credentials: 'include',
			});
			if (!optRes.ok) throw new Error('Failed to get options');
			const options = await optRes.json();

			const regResp = await startRegistration({ optionsJSON: options });

			const verifyRes = await fetch(`/api/auth/device/verify`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					deviceName: newDeviceName.trim() || undefined,
					response: regResp,
				}),
			});
			if (!verifyRes.ok) {
				const { error: err } = await verifyRes.json();
				throw new Error(err || 'Enrollment failed');
			}

			newDeviceName = '';
			success = 'Passkey enrolled successfully.';
			await loadDevices();
		} catch (err: any) {
			if (err?.name !== 'NotAllowedError') {
				error = err.message || 'Enrollment failed';
			}
		} finally {
			enrolling = false;
		}
	}

	async function remove(id: string) {
		error = '';
		success = '';
		const res = await fetch(`/api/auth/devices/${id}`, {
			method: 'DELETE',
			credentials: 'include',
		});
		if (!res.ok) {
			const { error: err } = await res.json();
			error = err || 'Failed to remove passkey';
			return;
		}
		await loadDevices();
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'medium' });
	}

	onMount(loadDevices);
</script>

<svelte:head>
	<title>Settings — Thufir</title>
</svelte:head>

<div class="max-w-xl mx-auto p-6 space-y-8">
	<h1 class="text-2xl font-bold text-gray-900">Settings</h1>

	<section class="space-y-4">
		<h2 class="text-lg font-semibold text-gray-800">Passkeys</h2>

		{#if loading}
			<div class="flex items-center gap-2 text-sm text-gray-500">
				<div class="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
				Loading…
			</div>
		{:else}
			<ul class="divide-y divide-gray-200 border border-gray-200 rounded-lg overflow-hidden">
				{#each devices as device}
					<li class="flex items-center justify-between px-4 py-3 bg-white">
						<div>
							<p class="text-sm font-medium text-gray-900">
								{device.device_name ?? 'Unnamed passkey'}
							</p>
							<p class="text-xs text-gray-400">Added {formatDate(device.created_at)}</p>
						</div>
						<button
							onclick={() => remove(device.id)}
							disabled={devices.length <= 1}
							title={devices.length <= 1 ? 'Cannot remove the last passkey' : 'Remove passkey'}
							class="text-sm text-red-500 hover:text-red-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
						>
							Remove
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		<div class="bg-gray-50 border border-gray-200 rounded-lg p-4 space-y-3">
			<p class="text-sm font-medium text-gray-700">Enroll a new passkey</p>
			<input
				type="text"
				bind:value={newDeviceName}
				placeholder="Device name (optional, e.g. iPhone)"
				class="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
			<button
				onclick={enroll}
				disabled={enrolling}
				class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
			>
				{#if enrolling}
					<div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
					Waiting for passkey…
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Enroll passkey
				{/if}
			</button>
		</div>

		{#if error}
			<p class="text-sm text-red-600">{error}</p>
		{/if}
		{#if success}
			<p class="text-sm text-green-600">{success}</p>
		{/if}
	</section>

	<section class="space-y-4">
		<h2 class="text-lg font-semibold text-gray-800">Bookmarklet</h2>
		<p class="text-sm text-gray-600">
			Drag the button below to your browser toolbar. Clicking it on any page will save
			that page's title and URL as a new task in your Thufir inbox.
		</p>
		<div class="flex items-center gap-4 bg-gray-50 border border-gray-200 rounded-lg p-4">
			<a
				href={bookmarkletHref}
				onclick={(e) => e.preventDefault()}
				draggable="true"
				class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg cursor-grab active:cursor-grabbing select-none hover:bg-blue-700 transition-colors"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				Add to Thufir
			</a>
			<p class="text-xs text-gray-500">Drag to your bookmarks toolbar</p>
		</div>
	</section>
</div>
