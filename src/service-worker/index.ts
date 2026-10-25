import { immutable, assets } from '$app/manifest';
import { version } from '$app/env';

const sw = self as unknown as ServiceWorkerGlobalScope;

const CACHE = `thufir-${version}`;
// The Go server answers "/" with index.html (200); every other SPA route gets
// the same shell with a 404 status, so the shell must be cached under "/".
const SHELL = new URL('./', sw.registration.scope).href;
const PRECACHE = [
	SHELL,
	...[...immutable, ...assets].map(({ path }) => new URL(path, sw.registration.scope).href)
];

sw.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(CACHE)
			.then((cache) => cache.addAll(PRECACHE))
			.then(() => sw.skipWaiting())
	);
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
			.then(() => sw.clients.claim())
	);
});

sw.addEventListener('fetch', (event) => {
	const request = event.request;
	if (request.method !== 'GET') return;
	const url = new URL(request.url);
	if (url.origin !== sw.location.origin || url.pathname.startsWith('/api/')) return;

	// Cache-first: the app shell must load with no network. New deploys arrive
	// as a new service worker version (see the controllerchange reload in
	// +layout.svelte).
	const key = request.mode === 'navigate' ? SHELL : request;
	event.respondWith(caches.match(key).then((cached) => cached ?? fetch(request)));
});
