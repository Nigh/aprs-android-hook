<script>
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	import '../app.css';
	import LanguageSelect from '$lib/LanguageSelect.svelte';
	import { initializeLocale } from '$lib/i18n.js';

	onMount(async () => {
		initializeLocale();
		const { registerSW } = await import('virtual:pwa-register');
		registerSW({ immediate: true });
	});
</script>

<svelte:head>
	<link rel="manifest" href={`${base}/manifest.webmanifest`} />
</svelte:head>

<div class="language-toolbar"><LanguageSelect /></div>
<slot />
