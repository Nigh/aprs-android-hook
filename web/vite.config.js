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
				name: 'APRS 轨迹',
				short_name: 'APRS 轨迹',
				description: 'aprs-android 私有定位记录与地图',
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
