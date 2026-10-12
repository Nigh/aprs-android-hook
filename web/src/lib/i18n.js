import { derived, writable } from 'svelte/store';
import { defaultLocale, resolveLocale, translate } from './locales/index.js';

export const locale = writable(defaultLocale);
export const t = derived(locale, (code) => (key, values) => translate(code, key, values));

export function setLocale(code) {
	const resolved = resolveLocale(code);
	locale.set(resolved);
	document.documentElement.lang = resolved;
	try {
		window.localStorage.setItem('aprs-language', resolved);
	} catch {
		/* Storage may be disabled. */
	}
}

export function initializeLocale() {
	let saved;
	try {
		saved = window.localStorage.getItem('aprs-language');
	} catch {
		/* Use English without persistence. */
	}
	setLocale(saved);
}
