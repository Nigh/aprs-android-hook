import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import prettier from 'eslint-config-prettier';

export default [
	{ ignores: ['.svelte-kit/**', 'build/**', 'node_modules/**'] },
	js.configs.recommended,
	...svelte.configs.recommended,
	{
		files: ['src/**/*.{js,svelte}'],
		languageOptions: {
			globals: Object.fromEntries(
				[
					'window',
					'document',
					'fetch',
					'Event',
					'URLSearchParams',
					'getComputedStyle',
					'setTimeout',
					'clearTimeout'
				].map((name) => [name, 'readonly'])
			)
		}
	},
	prettier
];
