import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { URL } from 'node:url';
import { locales, resolveLocale, translate } from '../src/lib/locales/index.js';
import { api } from '../src/lib/api.js';

test('catalogs cover the same keys and interpolation parameters', () => {
	const keys = Object.keys(locales.en.messages).sort();
	for (const language of Object.values(locales)) {
		assert.deepEqual(Object.keys(language.messages).sort(), keys);
		for (const key of keys) {
			const placeholders = (value) =>
				[...new Set(JSON.stringify(value).match(/\{\w+\}/g) || [])].sort();
			assert.deepEqual(
				placeholders(language.messages[key]),
				placeholders(locales.en.messages[key]),
				key
			);
		}
	}
	const codes = readFileSync(new URL('../../errors.go', import.meta.url), 'utf8').match(
		/"error\.\w+"/g
	);
	for (const code of codes) assert.ok(Object.hasOwn(locales.en.messages, JSON.parse(code)));
});

test('registry supports another language, English fallback, and native plural rules', () => {
	assert.equal(resolveLocale('ZH-cn'), 'zh-CN');
	assert.equal(resolveLocale('unknown'), 'en');
	assert.equal(translate('en', 'point.count', { count: 1 }), '1 point');
	assert.equal(translate('en', 'point.count', { count: 2 }), '2 points');
	assert.equal(translate('en', 'point.details', { count: 1 }), 'Location details (1 point)');
	assert.equal(translate('zh-CN', 'point.count', { count: 2 }), '2 个定位点');
	assert.equal(translate('en', 'tokens.noteLabel', { token: '<phone>' }), 'Note for <phone>');
	locales.es = { name: 'Español', messages: { language: 'Idioma' } };
	locales.ru = { name: 'Русский', messages: { language: 'Язык' } };
	try {
		assert.equal(resolveLocale('es'), 'es');
		assert.equal(translate('es', 'language'), 'Idioma');
		assert.equal(translate('es', 'auth.login'), 'Sign in');
		assert.equal(translate('ru', 'point.count', { count: 21 }), '21 points');
	} finally {
		delete locales.es;
		delete locales.ru;
	}
});

test('API errors expose a translation key and safely fall back for unknown errors', async () => {
	const original = globalThis.fetch;
	try {
		globalThis.fetch = async () => ({
			ok: false,
			status: 409,
			json: async () => ({ code: 'error.tokenLimit', error: '最多登记16个 token' })
		});
		await assert.rejects(api('tokens'), (error) => error.code === 'error.tokenLimit');
		globalThis.fetch = async () => ({
			ok: false,
			status: 500,
			json: async () => ({ code: 'unknown', error: 'internal detail' })
		});
		await assert.rejects(api('tokens'), (error) => error.code === 'error.requestFailed');
		globalThis.fetch = async () => {
			throw new Error('offline');
		};
		await assert.rejects(api('tokens'), (error) => error.code === 'error.network');
	} finally {
		globalThis.fetch = original;
	}
});
