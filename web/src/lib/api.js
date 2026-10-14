import { defaultLocale, locales, translate } from './locales/index.js';

export async function api(path, options = {}) {
	let response;
	try {
		response = await fetch(`/api/${path}`, {
			...options,
			headers: { 'Content-Type': 'application/json', ...options.headers },
			cache: 'no-store'
		});
	} catch (error) {
		if (error.name === 'AbortError') throw error;
		throw Object.assign(new Error('Network request failed', { cause: error }), {
			code: 'error.network'
		});
	}
	const value = await response.json();
	if (!response.ok) {
		if (response.status === 401 && !['login', 'register'].includes(path)) {
			window.dispatchEvent(new Event('session-expired'));
		}
		const code = Object.hasOwn(locales[defaultLocale].messages, value.code)
			? value.code
			: 'error.requestFailed';
		throw Object.assign(new Error(value.error || 'Request failed'), { code });
	}
	return value;
}

export function formatDistance(meters, language = defaultLocale) {
	const kilometers = meters >= 1000;
	const number = new Intl.NumberFormat(language, {
		minimumFractionDigits: kilometers ? 2 : 0,
		maximumFractionDigits: kilometers ? 2 : 0
	});
	return `${number.format(kilometers ? meters / 1000 : meters)} ${kilometers ? 'km' : 'm'}`;
}
export function formatDuration(ms, language = defaultLocale) {
	const minutes = Math.floor(ms / 60000);
	return minutes >= 60
		? translate(language, 'format.hoursMinutes', {
				hours: Math.floor(minutes / 60),
				minutes: minutes % 60
			})
		: translate(language, 'format.minutes', { minutes });
}
export function formatTime(ms, timezone, language = defaultLocale) {
	return new Intl.DateTimeFormat(language, {
		timeZone: timezone,
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23'
	}).format(ms);
}

export function formatMeasurement(value, unit, language = defaultLocale, unknown = '—') {
	if (value == null) return unknown;
	return `${new Intl.NumberFormat(language, { maximumFractionDigits: 2 }).format(value)}${unit === '°' ? '' : ' '}${unit}`;
}

export function shiftDate(date, days) {
	const d = new Date(`${date}T12:00:00Z`);
	d.setUTCDate(d.getUTCDate() + days);
	return d.toISOString().slice(0, 10);
}
