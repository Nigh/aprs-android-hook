import en from './en.json' with { type: 'json' };
import zhCN from './zh-CN.json' with { type: 'json' };

export const defaultLocale = 'en';
export const locales = {
	en: { name: 'English', messages: en },
	'zh-CN': { name: '简体中文', messages: zhCN }
};

export function resolveLocale(code) {
	return (
		Object.keys(locales).find(
			(candidate) => candidate.toLowerCase() === String(code).toLowerCase()
		) || defaultLocale
	);
}

export function translate(code, key, values = {}) {
	const language = resolveLocale(code);
	const source =
		Object.hasOwn(locales[language].messages, key) && locales[language].messages[key] != null
			? language
			: defaultLocale;
	let message = locales[source].messages[key] ?? key;
	if (typeof message === 'object') {
		message = message[new Intl.PluralRules(source).select(values.count)] ?? message.other;
	}
	return message.replace(/\{(\w+)\}/g, (placeholder, name) =>
		values[name] == null ? placeholder : String(values[name])
	);
}
