import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		SvelteKitPWA({
			registerType: 'autoUpdate',
			workbox: {
				navigateFallback: '/',
				navigateFallbackDenylist: [/^\/api\//],
				globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
				// Only the shell is precached; private APIs and map tiles stay network-only.
				runtimeCaching: []
			},
			manifest: {
				name: 'APRS Tracks',
				short_name: 'APRS Tracks',
				description: 'Private aprs-android location history and maps',
				lang: 'en',
				scope: '/',
				start_url: '/',
				background_color: '#212121',
				theme_color: '#212121',
				display: 'standalone',
				icons: [{ src: 'icon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }]
			}
		})
	],
	server: { proxy: { '/api': 'http://127.0.0.1:8080' } }
});
