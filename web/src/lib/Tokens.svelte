<script>
	import { onMount } from 'svelte';
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
			.catch((e) => (error = e.message))
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
			error = e.message;
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
		}, 'token 已登记，可在手机上启用 webhook');
	}
	function askDelete(token) {
		pending = token;
		dialog.showModal();
	}
	function remove() {
		const token = pending;
		dialog.close();
		perform(
			() => api(`tokens/${token.key}`, { method: 'DELETE' }),
			'token 已删除，历史定位记录已保留'
		);
	}
</script>

<section class="token-page" aria-labelledby="tokens-title">
	<div class="section-heading">
		<h2 id="tokens-title">上传 token</h2>
		<span class="badge badge-outline">{tokens.length} / 16</span>
	</div>
	<p class="muted">
		在 aprs-android 的设置 → Webhook 中复制 token，粘贴到这里登记。只有已登记的 token 可以上传定位。
	</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	{#if message}<p class="notice success" role="status">{message}</p>{/if}
	<form class="token-form" onsubmit={add}>
		<label class="field"
			><span>手机 token</span><input
				class="input"
				type="password"
				bind:value
				autocomplete="off"
				required
				pattern={'[0-9a-f]{64}'}
				minlength="64"
				maxlength="64"
				placeholder="64 位小写十六进制字符"
				disabled={busy || tokens.length >= 16}
			/></label
		>
		<label class="field"
			><span>备注 <span class="muted">（可选）</span></span><input
				class="input"
				bind:value={note}
				maxlength="128"
				placeholder="例如：日常使用的手机"
				disabled={busy}
			/></label
		>
		<button class="btn btn-primary" disabled={busy || tokens.length >= 16 || loading}
			>{busy ? '正在保存…' : '登记 token'}</button
		>
	</form>
	<p class="muted small">
		服务端仅保存 token 哈希，保存后无法查看完整值。手机刷新 token 后，请删除旧记录并登记新值。
	</p>
	{#if loading}<p role="status">正在读取 token…</p>
	{:else if !tokens.length}<div class="empty">
			<h3>还没有登记 token</h3>
			<p>先复制手机 token，再填写上方表单。</p>
		</div>
	{:else}<div class="token-list">
			{#each tokens as token (token.key)}<div class="token-row">
					<div>
						<code>{token.mask}</code>
						<p class="muted small">
							最近上传：{token.last_upload_ms
								? formatTime(token.last_upload_ms, timezone)
								: '尚未上传'}
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
								'备注已保存'
							);
						}}
					>
						<label class="field"
							><span class="sr-only">{token.mask} 的备注</span><input
								class="input"
								bind:value={notes[token.key]}
								maxlength="128"
								placeholder="添加备注"
								disabled={busy}
							/></label
						>
						<button class="btn btn-sm btn-ghost" disabled={busy || notes[token.key] === token.note}
							>保存备注</button
						>
					</form>
					<button
						class="btn btn-sm btn-ghost danger"
						onclick={() => askDelete(token)}
						disabled={busy}>删除</button
					>
				</div>{/each}
		</div>{/if}
</section>
<dialog class="modal" bind:this={dialog} aria-labelledby="delete-token-title">
	<div class="modal-box">
		<h3 id="delete-token-title">删除这个 token？</h3>
		<p>删除后，使用 {pending?.mask} 的设备将无法上传。历史定位记录会保留。</p>
		<div class="modal-action">
			<button class="btn" onclick={() => dialog.close()}>取消</button><button
				class="btn btn-error"
				onclick={remove}>删除 token</button
			>
		</div>
	</div>
</dialog>
