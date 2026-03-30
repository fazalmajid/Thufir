<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { startRegistration, startAuthentication } from '@simplewebauthn/browser';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

	type Mode = 'loading' | 'login' | 'setup';

	let mode = $state<Mode>('loading');
	let displayName = $state('');
	let deviceName = $state('');
	let error = $state('');
	let busy = $state(false);

	onMount(async () => {
		const res = await fetch(`${API_URL}/api/auth/status`);
		const { hasUsers } = await res.json();
		mode = hasUsers ? 'login' : 'setup';
	});

	async function login() {
		busy = true;
		error = '';
		try {
			const optRes = await fetch(`${API_URL}/api/auth/login/options`, {
				method: 'POST',
				credentials: 'include',
			});
			if (!optRes.ok) throw new Error('Failed to get login options');
			const options = await optRes.json();

			const authResp = await startAuthentication({ optionsJSON: options });

			const verifyRes = await fetch(`${API_URL}/api/auth/login/verify`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ response: authResp }),
			});
			if (!verifyRes.ok) {
				const { error: err } = await verifyRes.json();
				throw new Error(err || 'Authentication failed');
			}

			goto('/inbox');
		} catch (err: any) {
			// User cancelled the passkey prompt — don't show an error
			if (err?.name === 'NotAllowedError') {
				error = '';
			} else {
				error = err.message || 'Authentication failed';
			}
		} finally {
			busy = false;
		}
	}

	async function setup() {
		if (!displayName.trim()) {
			error = 'Please enter your name';
			return;
		}
		busy = true;
		error = '';
		try {
			const optRes = await fetch(`${API_URL}/api/auth/setup/options`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ displayName: displayName.trim() }),
			});
			if (!optRes.ok) {
				const { error: err } = await optRes.json();
				throw new Error(err || 'Setup failed');
			}
			const { options, userId } = await optRes.json();

			const regResp = await startRegistration({ optionsJSON: options });

			const verifyRes = await fetch(`${API_URL}/api/auth/setup/verify`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					userId,
					displayName: displayName.trim(),
					deviceName: deviceName.trim() || undefined,
					response: regResp,
				}),
			});
			if (!verifyRes.ok) {
				const { error: err } = await verifyRes.json();
				throw new Error(err || 'Setup failed');
			}

			goto('/inbox');
		} catch (err: any) {
			if (err?.name === 'NotAllowedError') {
				error = '';
			} else {
				error = err.message || 'Setup failed';
			}
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Thufir — Sign in</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
	<div class="w-full max-w-sm">
		<div class="bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
			<div class="mb-8 text-center">
				<h1 class="text-2xl font-bold text-gray-900">Thufir</h1>
				<p class="text-sm text-gray-500 mt-1">Local-first tasks</p>
			</div>

			{#if mode === 'loading'}
				<div class="flex justify-center py-8">
					<div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
				</div>

			{:else if mode === 'login'}
				<div class="space-y-4">
					<p class="text-sm text-gray-600 text-center">
						Sign in with your passkey — your device will prompt you to authenticate.
					</p>

					<button
						onclick={login}
						disabled={busy}
						class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 active:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						{#if busy}
							<div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
							Waiting for passkey…
						{:else}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
							</svg>
							Sign in with passkey
						{/if}
					</button>
				</div>

			{:else if mode === 'setup'}
				<div class="space-y-4">
					<p class="text-sm text-gray-600 text-center">
						Welcome! Create your account by registering a passkey on this device.
					</p>

					<div>
						<label for="displayName" class="block text-sm font-medium text-gray-700 mb-1">
							Your name
						</label>
						<input
							id="displayName"
							type="text"
							bind:value={displayName}
							placeholder="e.g. Majid"
							autocomplete="name"
							class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						/>
					</div>

					<div>
						<label for="deviceName" class="block text-sm font-medium text-gray-700 mb-1">
							Device name <span class="text-gray-400 font-normal">(optional)</span>
						</label>
						<input
							id="deviceName"
							type="text"
							bind:value={deviceName}
							placeholder="e.g. MacBook Touch ID"
							class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						/>
					</div>

					<button
						onclick={setup}
						disabled={busy}
						class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 active:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						{#if busy}
							<div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
							Waiting for passkey…
						{:else}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
							</svg>
							Create account with passkey
						{/if}
					</button>
				</div>
			{/if}

			{#if error}
				<p class="mt-4 text-sm text-red-600 text-center">{error}</p>
			{/if}
		</div>

		<p class="mt-4 text-xs text-center text-gray-400">
			Passkeys are stored on your device. No password needed.
		</p>
	</div>
</div>
