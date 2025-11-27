import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit(),
		SvelteKitPWA({
			srcDir: './src',
			mode: 'development',
			strategies: 'injectManifest',
			filename: 'service-worker.ts',
			manifest: {
				name: 'Thufir - Personal Task Manager',
				short_name: 'Thufir',
				description: 'Local-first task management inspired by Things',
				theme_color: '#3b82f6',
				background_color: '#ffffff',
				display: 'standalone',
				start_url: '/',
				scope: '/',
				icons: [
					{
						src: '/icons/icon-192x192.png',
						sizes: '192x192',
						type: 'image/png'
					},
					{
						src: '/icons/icon-512x512.png',
						sizes: '512x512',
						type: 'image/png',
						purpose: 'any maskable'
					}
				],
				categories: ['productivity', 'utilities']
			},
			devOptions: {
				enabled: true,
				type: 'module'
			}
		})
	]
});
