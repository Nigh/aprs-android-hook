<script>
	import { onMount, tick } from 'svelte';
	import { locale, t } from '$lib/i18n.js';
	import {
		api,
		formatDistance,
		formatDuration,
		formatMeasurement,
		formatTime,
		shiftDate
	} from '$lib/api.js';
	import TrackMap from '$lib/TrackMap.svelte';
	import Tokens from '$lib/Tokens.svelte';
	import { blankClick } from '$lib/blankClick.js';
	let state = $state(null),
		error = $state(''),
		username = $state(''),
		password = $state(''),
		busy = $state(false);
	let view = $state('tracks'),
		from = $state(''),
		to = $state(''),
		preset = $state('today'),
		id = $state(''),
		device = $state('');
	let sources = $state([]),
		data = $state(null),
		paths = $state([]),
		selected = $state(null),
		selectedKey = $state('');
	let detailLoading = $state(false),
		loading = $state(false),
		page = $state(1);
	let applied = '',
		requestVersion = 0,
		detailVersion = 0;
	let tabs = $state();
	let trackRows = $state();
	const ids = $derived([...new Set(sources.map((source) => source.id))]);
	const devices = $derived([
		...new Set(
			sources.filter((source) => !id || source.id === id).map((source) => source.device_hash)
		)
	]);
	async function initialize() {
		error = '';
		try {
			state = await api('auth');
			from = to = state.today;
			if (state.authenticated) await reload();
		} catch (e) {
			error = e.code || 'error.requestFailed';
		}
	}
	function expire() {
		requestVersion++;
		detailVersion++;
		state = state ? { ...state, authenticated: false } : null;
		data = selected = null;
		paths = [];
		sources = [];
		view = 'tracks';
		error = 'error.sessionExpired';
	}
	onMount(() => {
		initialize();
		window.addEventListener('session-expired', expire);
		return () => {
			requestVersion++;
			detailVersion++;
			window.removeEventListener('session-expired', expire);
		};
	});
	async function authenticate(event) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			await api(state.registered ? 'login' : 'register', {
				method: 'POST',
				body: JSON.stringify({ username, password })
			});
			password = '';
			view = 'tracks';
			await initialize();
		} catch (e) {
			error = e.code || 'error.requestFailed';
		} finally {
			busy = false;
		}
	}
	async function logout() {
		busy = true;
		error = '';
		try {
			await api('logout', { method: 'POST' });
			expire();
			error = '';
		} catch (e) {
			error = e.code || 'error.requestFailed';
		} finally {
			busy = false;
		}
	}
	function changePreset(value) {
		preset = value;
		if (value === 'custom') return;
		const days = { today: 0, yesterday: 1, week: 6, month: 29 };
		to = value === 'yesterday' ? shiftDate(state.today, -1) : state.today;
		from = shiftDate(state.today, -days[value]);
	}
	async function reload() {
		if (!from || !to || from > to) {
			error = 'error.invalidRange';
			return;
		}
		const version = ++requestVersion;
		detailVersion++;
		loading = true;
		error = '';
		selected = null;
		selectedKey = '';
		data = null;
		paths = [];
		const query = new URLSearchParams({ from, to, id, device }).toString();
		try {
			const [result, available] = await Promise.all([api(`tracks?${query}&map=1`), api('sources')]);
			if (version !== requestVersion) return;
			data = result;
			paths = result.paths || [];
			sources = available;
			page = 1;
			applied = query;
			if (data.tracks.length) await choose(data.tracks[0]);
		} catch (e) {
			if (version === requestVersion) error = e.code || 'error.requestFailed';
		} finally {
			if (version === requestVersion) loading = false;
		}
	}
	function changePage(nextPage) {
		page = nextPage;
		data = { ...data, tracks: data.all_tracks.slice((page - 1) * 50, page * 50) };
	}
	function deselect() {
		detailVersion++;
		selected = null;
		selectedKey = '';
		detailLoading = false;
	}
	function choosePath(key, targetPage) {
		choose({ key });
		if (targetPage !== page) changePage(targetPage);
		tick().then(() => trackRows?.querySelector('.selected')?.scrollIntoView({ block: 'nearest' }));
	}
	async function choose(track) {
		const version = ++detailVersion;
		selectedKey = track.key;
		selected = null;
		detailLoading = true;
		error = '';
		try {
			const detail = await api(`tracks/${encodeURIComponent(track.key)}?${applied}`);
			if (version === detailVersion) selected = detail;
		} catch (e) {
			if (version === detailVersion) error = e.code || 'error.requestFailed';
		} finally {
			if (version === detailVersion) detailLoading = false;
		}
	}
	function tabKey(event) {
		if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
		event.preventDefault();
		view =
			event.key === 'Home'
				? 'tracks'
				: event.key === 'End'
					? 'tokens'
					: view === 'tracks'
						? 'tokens'
						: 'tracks';
		tabs.querySelector(`[id="tab-${view}"]`).focus();
	}
</script>

<svelte:head
	><title>{$t('app.name')} · aprs-android-hook</title><meta
		name="description"
		content={$t('app.description')}
	/></svelte:head
>
{#if !state}
	<main class="auth-shell">
		<div class="auth-panel">
			<h1>{$t('app.name')}</h1>
			{#if error}<p class="notice error" role="alert">{$t(error)}</p>
				<button class="btn btn-primary" onclick={initialize}>{$t('app.reconnect')}</button>{:else}<p
					role="status"
				>
					{$t('app.connecting')}
				</p>{/if}
		</div>
	</main>
{:else if !state.authenticated}
	<main class="auth-shell">
		<div class="auth-panel">
			<div class="brand-mark" aria-hidden="true">
				<svg viewBox="0 0 48 48"><path d="m8 36 10-24 14 18 9-20" /></svg>
			</div>
			<h1>{state.registered ? $t('auth.loginTitle') : $t('auth.createTitle')}</h1>
			<p class="muted">
				{state.registered ? $t('auth.loginHelp') : $t('auth.createHelp')}
			</p>
			{#if error}<p class="notice error" role="alert">{$t(error)}</p>{/if}
			<form class="auth-form" onsubmit={authenticate}>
				<label class="field"
					><span>{$t('auth.username')}</span><input
						class="input"
						bind:value={username}
						autocomplete="username"
						required
						maxlength="64"
						disabled={busy}
					/></label
				>
				<label class="field"
					><span>{$t('auth.password')}</span><input
						class="input"
						type="password"
						bind:value={password}
						autocomplete={state.registered ? 'current-password' : 'new-password'}
						required
						minlength={state.registered ? 1 : 12}
						disabled={busy}
					/>{#if !state.registered}<span class="field-help muted small"
							>{$t('auth.passwordHelp')}</span
						>{/if}</label
				>
				<button class="btn btn-primary" disabled={busy}
					>{busy
						? $t('auth.wait')
						: state.registered
							? $t('auth.login')
							: $t('auth.create')}</button
				>
			</form>
			<p class="auth-footer muted small">aprs-android-hook · {$t('auth.privateHistory')}</p>
		</div>
	</main>
{:else}
	<div class="app-shell">
		<header class="app-header">
			<div>
				<h1>{$t('app.name')}</h1>
				<p class="muted small">{$t('app.subtitle')}</p>
			</div>
			<button class="btn btn-sm btn-ghost" disabled={busy} onclick={logout}
				>{$t('auth.logout')}</button
			>
		</header>
		<div
			class="tabs app-tabs"
			role="tablist"
			tabindex="-1"
			aria-label={$t('app.navigation')}
			bind:this={tabs}
			onkeydown={tabKey}
		>
			<button
				class="tab"
				role="tab"
				id="tab-tracks"
				aria-selected={view === 'tracks'}
				aria-controls="panel-tracks"
				tabindex={view === 'tracks' ? 0 : -1}
				onclick={() => (view = 'tracks')}>{$t('tracks.tab')}</button
			>
			<button
				class="tab"
				role="tab"
				id="tab-tokens"
				aria-selected={view === 'tokens'}
				aria-controls="panel-tokens"
				tabindex={view === 'tokens' ? 0 : -1}
				onclick={() => (view = 'tokens')}>{$t('tokens.tab')}</button
			>
		</div>
		<main>
			{#if error}<p class="notice error" role="alert">{$t(error)}</p>{/if}
			{#if view === 'tokens'}<div role="tabpanel" id="panel-tokens" aria-labelledby="tab-tokens">
					<Tokens timezone={state.timezone} />
				</div>
			{:else}<div role="tabpanel" id="panel-tracks" aria-labelledby="tab-tracks">
					<form
						class="filters"
						onsubmit={(event) => {
							event.preventDefault();
							reload();
						}}
					>
						<label class="field"
							><span>{$t('filter.range')}</span><select
								class="select"
								value={preset}
								onchange={(event) => changePreset(event.currentTarget.value)}
								><option value="today">{$t('filter.today')}</option><option value="yesterday"
									>{$t('filter.yesterday')}</option
								><option value="week">{$t('filter.week')}</option><option value="month"
									>{$t('filter.month')}</option
								><option value="custom">{$t('filter.custom')}</option></select
							></label
						>
						<label class="field"
							><span>{$t('filter.from')}</span><input
								class="input"
								type="date"
								bind:value={from}
								required
								onchange={() => (preset = 'custom')}
							/></label
						>
						<label class="field"
							><span>{$t('filter.to')}</span><input
								class="input"
								type="date"
								bind:value={to}
								min={from}
								required
								onchange={() => (preset = 'custom')}
							/></label
						>
						<label class="field"
							><span>{$t('filter.id')}</span><select
								class="select"
								bind:value={id}
								onchange={() => (device = '')}
								><option value="">{$t('filter.allIDs')}</option
								>{#each ids as sourceID (sourceID)}<option value={sourceID}>{sourceID}</option
									>{/each}</select
							></label
						>
						<label class="field"
							><span>{$t('filter.device')}</span><select
								class="select device-select"
								bind:value={device}
								><option value="">{$t('filter.allDevices')}</option
								>{#each devices as hash (hash)}<option value={hash}>{hash}</option>{/each}</select
							></label
						>
						<button class="btn btn-primary" disabled={loading}
							>{loading ? $t('tracks.querying') : $t('tracks.show')}</button
						>
					</form>
					<div class="summary" aria-live="polite">
						<div>
							<span class="muted">{$t('tracks.count')}</span><strong
								>{data?.summary.track_count ?? '—'}</strong
							>
						</div>
						<div>
							<span class="muted">{$t('tracks.distance')}</span><strong
								>{data ? formatDistance(data.summary.distance_m, $locale) : '—'}</strong
							>
						</div>
						<div>
							<span class="muted">{$t('tracks.duration')}</span><strong
								>{data ? formatDuration(data.summary.duration_ms, $locale) : '—'}</strong
							>
						</div>
						<p class="muted small">
							{$t('timezone', { timezone: state.timezone })}<br />{$t('tracks.rule')}
						</p>
					</div>
					<div class="workspace">
						<section
							class="track-list"
							aria-label={$t('tracks.list')}
							aria-busy={loading}
							use:blankClick={deselect}
						>
							<div class="list-heading">
								<h2>{$t('tracks.list')}</h2>
								<span class="muted small"
									>{$t('point.count', { count: data?.summary.point_count ?? 0 })}</span
								>
							</div>
							{#if loading}<div class="empty" role="status">{$t('tracks.loading')}</div>
							{:else if !data}<div class="empty">
									<p>{$t('tracks.failed')}</p>
									<button class="btn btn-sm" onclick={() => reload()}>{$t('common.retry')}</button>
								</div>
							{:else if !data.tracks.length}<div class="empty">
									<h3>{$t('tracks.empty')}</h3>
									<p class="muted">{$t('tracks.emptyHelp')}</p>
									<button class="btn btn-sm" onclick={() => (view = 'tokens')}
										>{$t('tracks.openSettings')}</button
									>
								</div>
							{:else}
								<div class="track-rows" bind:this={trackRows}>
									{#each data.tracks as track (track.key)}<button
											class="track-row"
											class:selected={selectedKey === track.key}
											aria-pressed={selectedKey === track.key}
											onclick={() => choose(track)}
											><span class="track-identity"
												><span class="track-id">{track.id}</span><span
													class="muted device-hash"
													title={$t('device.label', { device: track.device_hash })}
													>{$t('device.label', { device: track.device_hash })}</span
												></span
											><span class="muted small"
												>{formatTime(track.start_ms, state.timezone, $locale)} — {formatTime(
													track.end_ms,
													state.timezone,
													$locale
												)}</span
											><span class="track-measures"
												><span>{formatDistance(track.distance_m, $locale)}</span><span
													>{formatDuration(track.duration_ms, $locale)}</span
												><span>{$t('point.count', { count: track.point_count })}</span></span
											></button
										>{/each}
								</div>
								<div class="pagination">
									<button
										class="btn btn-sm btn-ghost"
										disabled={page <= 1 || loading}
										onclick={() => changePage(page - 1)}>{$t('tracks.previous')}</button
									><span class="small"
										>{page} / {Math.max(1, Math.ceil(data.summary.track_count / 50))}</span
									><button
										class="btn btn-sm btn-ghost"
										disabled={page * 50 >= data.summary.track_count || loading}
										onclick={() => changePage(page + 1)}>{$t('tracks.next')}</button
									>
								</div>
							{/if}
						</section>
						<section class="map-panel" aria-label={$t('tracks.details')}>
							<div class="map-heading">
								<h2>
									{selected ? selected.id : detailLoading ? $t('tracks.loading') : $t('map.title')}
								</h2>
								<span class="muted small"
									>{selected
										? `${$t('point.count', { count: selected.point_count })} · ${formatDistance(selected.distance_m, $locale)}`
										: $t('tracks.select')}</span
								>
							</div>
							<TrackMap
								track={selected}
								{paths}
								onSelect={choosePath}
								onDeselect={deselect}
								timezone={state.timezone}
							/>
							{#if selected}<details class="point-details">
									<summary>{$t('point.details', { count: selected.point_count })}</summary>
									<div class="point-table-scroll">
										<table class="table table-sm">
											<thead
												><tr
													><th>{$t('point.time')}</th><th>{$t('point.coordinates')}</th><th
														>{$t('point.accuracy')}</th
													><th>{$t('point.speed')}</th><th>{$t('point.altitude')}</th><th
														>{$t('point.bearing')}</th
													></tr
												></thead
											><tbody
												>{#each selected.points as point (point.timestamp_ms)}<tr
														><td>{formatTime(point.timestamp_ms, state.timezone, $locale)}</td><td
															>{point.latitude.toFixed(6)} / {point.longitude.toFixed(6)}</td
														><td>{formatMeasurement(point.accuracy_m, 'm', $locale)}</td><td
															>{formatMeasurement(point.speed_mps, 'm/s', $locale)}</td
														><td>{formatMeasurement(point.altitude_m, 'm', $locale)}</td><td
															>{formatMeasurement(point.bearing_deg, '°', $locale)}</td
														></tr
													>{/each}</tbody
											>
										</table>
									</div>
								</details>{/if}
						</section>
					</div>
				</div>{/if}
		</main>
		<footer class="app-footer muted small">
			aprs-android-hook · {$t('tracks.footer')}
		</footer>
	</div>
{/if}
