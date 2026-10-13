<script>
	import { onMount } from 'svelte';
	import { locale, t } from '$lib/i18n.js';
	import { api, formatTime } from './api.js';
	let { timezone } = $props();
	let tokens = $state([]);
	let notes = $state({});
	let value = $state('');
	let note = $state('');
	let error = $state('');
	let message = $state('');
	let busy = $state(false);
	let loading = $state(true);
	let pending = $state(null);
	let dialog;
	async function load() {
		tokens = await api('tokens');
		notes = Object.fromEntries(tokens.map((token) => [token.key, token.note]));
	}
	onMount(() => {
		load()
			.catch((e) => (error = e.code || 'error.requestFailed'))
			.finally(() => (loading = false));
	});
	async function perform(action, success) {
		busy = true;
		error = '';
		message = '';
		try {
			await action();
			await load();
			message = success;
		} catch (e) {
			error = e.code || 'error.requestFailed';
		} finally {
			busy = false;
		}
	}
	function add(event) {
		event.preventDefault();
		perform(async () => {
			await api('tokens', { method: 'POST', body: JSON.stringify({ token: value, note }) });
			value = '';
			note = '';
		}, 'success.tokenAdded');
	}
	function askDelete(token) {
		pending = token;
		dialog.showModal();
	}
	function remove() {
		const token = pending;
		dialog.close();
		perform(() => api(`tokens/${token.key}`, { method: 'DELETE' }), 'success.tokenDeleted');
	}
</script>

<section class="token-page" aria-labelledby="tokens-title">
	<div class="section-heading">
		<h2 id="tokens-title">{$t('tokens.title')}</h2>
		<span class="badge badge-outline">{tokens.length} / 16</span>
	</div>
	<p class="muted">
		{$t('tokens.help')}
	</p>
	{#if error}<p class="notice error" role="alert">{$t(error)}</p>{/if}
	{#if message}<p class="notice success" role="status">{$t(message)}</p>{/if}
	<form class="token-form" onsubmit={add}>
		<label class="field"
			><span>{$t('tokens.phone')}</span><input
				class="input"
				type="password"
				bind:value
				autocomplete="off"
				required
				pattern={'[0-9a-f]{64}'}
				minlength="64"
				maxlength="64"
				placeholder={$t('tokens.placeholder')}
				disabled={busy || tokens.length >= 16}
			/></label
		>
		<label class="field"
			><span>{$t('tokens.note')} <span class="muted">{$t('tokens.optional')}</span></span><input
				class="input"
				bind:value={note}
				maxlength="128"
				placeholder={$t('tokens.noteExample')}
				disabled={busy}
			/></label
		>
		<button class="btn btn-primary" disabled={busy || tokens.length >= 16 || loading}
			>{busy ? $t('tokens.saving') : $t('tokens.add')}</button
		>
	</form>
	<p class="muted small">
		{$t('tokens.hashHelp')}
	</p>
	{#if loading}<p role="status">{$t('tokens.loading')}</p>
	{:else if !tokens.length}<div class="empty">
			<h3>{$t('tokens.empty')}</h3>
			<p>{$t('tokens.emptyHelp')}</p>
		</div>
	{:else}<div class="token-list">
			{#each tokens as token (token.key)}<div class="token-row">
					<div>
						<code>{token.mask}</code>
						<p class="muted small">
							{$t('tokens.lastUpload')}{token.last_upload_ms
								? formatTime(token.last_upload_ms, timezone, $locale)
								: $t('tokens.never')}
						</p>
					</div>
					<form
						class="token-edit"
						onsubmit={(event) => {
							event.preventDefault();
							perform(
								() =>
									api(`tokens/${token.key}`, {
										method: 'PATCH',
										body: JSON.stringify({ note: notes[token.key] })
									}),
								'success.noteSaved'
							);
						}}
					>
						<label class="field"
							><span class="sr-only">{$t('tokens.noteLabel', { token: token.mask })}</span><input
								class="input"
								bind:value={notes[token.key]}
								maxlength="128"
								placeholder={$t('tokens.notePlaceholder')}
								disabled={busy}
							/></label
						>
						<button class="btn btn-sm btn-ghost" disabled={busy || notes[token.key] === token.note}
							>{$t('tokens.saveNote')}</button
						>
					</form>
					<button
						class="btn btn-sm btn-ghost danger"
						onclick={() => askDelete(token)}
						disabled={busy}>{$t('tokens.delete')}</button
					>
				</div>{/each}
		</div>{/if}
</section>
<dialog class="modal" bind:this={dialog} aria-labelledby="delete-token-title">
	<div class="modal-box">
		<h3 id="delete-token-title">{$t('tokens.deleteTitle')}</h3>
		<p>{$t('tokens.deleteHelp', { token: pending?.mask })}</p>
		<div class="modal-action">
			<button class="btn" onclick={() => dialog.close()}>{$t('common.cancel')}</button><button
				class="btn btn-error"
				onclick={remove}>{$t('tokens.confirmDelete')}</button
			>
		</div>
	</div>
</dialog>
