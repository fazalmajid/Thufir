import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: 'dist',
			assets: 'dist',
			fallback: 'index.html', // SPA mode — Go server handles all non-file paths
			precompress: false,
			strict: false
		}),
		// @vite-pwa/sveltekit owns service worker registration (see
		// +layout.svelte); this avoids SvelteKit's own auto-registration
		// stepping on it if a src/service-worker.js is ever added later.
		serviceWorker: {
			register: false
		}
	}
};

export default config;
