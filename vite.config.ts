import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit(),
		SvelteKitPWA({
			registerType: 'autoUpdate',
			manifest: {
				name: 'Thufir',
				short_name: 'Thufir',
				description: 'Local-first task manager',
				theme_color: '#3b82f6',
				background_color: '#ffffff',
				display: 'standalone',
				start_url: '/',
				icons: [
					{
						src: '/icon.svg',
						sizes: 'any',
						type: 'image/svg+xml',
						purpose: 'any',
					},
					{
						src: '/icon.svg',
						sizes: 'any',
						type: 'image/svg+xml',
						purpose: 'maskable',
					},
				],
			},
			workbox: {
				globPatterns: ['**/*.{js,css,html,svg,png,ico,woff,woff2}'],
				navigateFallback: '/',
			},
			devOptions: {
				enabled: false,
			},
		}),
	],
	preview: {
		allowedHosts: ['thufir.majid.org'],
	},
});
