<script>
	import { locale, t, setLocale } from './i18n.js';
	import { locales } from './locales/index.js';
</script>

<label class="field language-select">
	<span>{$t('language')}</span>
	<select
		class="select select-sm"
		value={$locale}
		onchange={(event) => setLocale(event.currentTarget.value)}
	>
		{#each Object.entries(locales) as [code, language] (code)}
			<option value={code} lang={code}>{language.name}</option>
		{/each}
	</select>
</label>
