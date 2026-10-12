import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter(),
		csp: {
			mode: 'hash',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'img-src': ['self', 'data:', 'blob:', 'https://tiles.openfreemap.org'],
				'connect-src': ['self', 'https://tiles.openfreemap.org'],
				'worker-src': ['self', 'blob:'],
				'font-src': ['self'],
				'base-uri': ['self'],
				'form-action': ['self']
			}
		},
		paths: { base: globalThis.process.env.BASE_PATH || '' }
	},
	preprocess: vitePreprocess()
};

export default config;
