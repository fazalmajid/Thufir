import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			preprocess: vitePreprocess(),
			adapter: adapter({
				pages: 'dist',
				assets: 'dist',
				fallback: 'index.html' /* SPA mode — Go server handles all non-file paths */,
				precompress: false,
				strict: false
			})
		})
	],
	preview: { allowedHosts: ['thufir.majid.org'] }
});
