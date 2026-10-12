<script>
	import { onMount } from 'svelte';
	import { api, formatDistance, formatDuration, formatTime, shiftDate } from '$lib/api.js';
	import TrackMap from '$lib/TrackMap.svelte';
	import Tokens from '$lib/Tokens.svelte';
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
		selected = $state(null),
		selectedKey = $state('');
	let detailLoading = $state(false),
		loading = $state(false),
		page = $state(1);
	let applied = '',
		requestVersion = 0,
		detailVersion = 0;
	let tabs = $state();
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
			error = e.message;
		}
	}
	function expire() {
		requestVersion++;
		detailVersion++;
		state = state ? { ...state, authenticated: false } : null;
		data = selected = null;
		sources = [];
		view = 'tracks';
		error = '登录已过期，请重新登录';
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
			error = e.message;
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
			error = e.message;
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
	async function reload(nextPage = 1) {
		if (!from || !to || from > to) {
			error = '请选择有效日期，结束日期不能早于开始日期';
			return;
		}
		const version = ++requestVersion;
		detailVersion++;
		loading = true;
		error = '';
		selected = null;
		selectedKey = '';
		data = null;
		const query = new URLSearchParams({ from, to, id, device }).toString();
		try {
			const [result, available] = await Promise.all([
				api(`tracks?${query}&page=${nextPage}`),
				api('sources')
			]);
			if (version !== requestVersion) return;
			data = result;
			sources = available;
			page = nextPage;
			applied = query;
			if (data.tracks.length) await choose(data.tracks[0]);
		} catch (e) {
			if (version === requestVersion) error = e.message;
		} finally {
			if (version === requestVersion) loading = false;
		}
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
			if (version === detailVersion) error = e.message;
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
	><title>APRS 轨迹 · aprs-android-hook</title><meta
		name="description"
		content="私有 aprs-android 定位记录、路径统计与地图"
	/></svelte:head
>
{#if !state}
	<main class="auth-shell">
		<div class="auth-panel">
			<h1>APRS 轨迹</h1>
			{#if error}<p class="notice error" role="alert">{error}</p>
				<button class="btn btn-primary" onclick={initialize}>重新连接</button>{:else}<p
					role="status"
				>
					正在连接服务…
				</p>{/if}
		</div>
	</main>
{:else if !state.authenticated}
	<main class="auth-shell">
		<div class="auth-panel">
			<div class="brand-mark" aria-hidden="true">
				<svg viewBox="0 0 48 48"><path d="m8 36 10-24 14 18 9-20" /></svg>
			</div>
			<h1>{state.registered ? '登录 APRS 轨迹' : '创建你的账户'}</h1>
			<p class="muted">
				{state.registered
					? '查看你的定位记录、路径和地图。'
					: '首个注册账户是此服务的唯一用户。注册后即可接入手机。'}
			</p>
			{#if error}<p class="notice error" role="alert">{error}</p>{/if}
			<form class="auth-form" onsubmit={authenticate}>
				<label class="field"
					><span>用户名</span><input
						class="input"
						bind:value={username}
						autocomplete="username"
						required
						maxlength="64"
						disabled={busy}
					/></label
				>
				<label class="field"
					><span>密码</span><input
						class="input"
						type="password"
						bind:value={password}
						autocomplete={state.registered ? 'current-password' : 'new-password'}
						required
						minlength={state.registered ? 1 : 12}
						disabled={busy}
					/>{#if !state.registered}<span class="field-help muted small"
							>至少12个字符，最多72字节；中文字符会占用多个字节。</span
						>{/if}</label
				>
				<button class="btn btn-primary" disabled={busy}
					>{busy ? '请稍候…' : state.registered ? '登录' : '创建账户'}</button
				>
			</form>
			<p class="auth-footer muted small">aprs-android-hook · 私有定位记录</p>
		</div>
	</main>
{:else}
	<div class="app-shell">
		<header class="app-header">
			<div>
				<h1>APRS 轨迹</h1>
				<p class="muted small">你的定位记录与行程</p>
			</div>
			<button class="btn btn-sm btn-ghost" disabled={busy} onclick={logout}>退出登录</button>
		</header>
		<div
			class="tabs app-tabs"
			role="tablist"
			tabindex="-1"
			aria-label="功能"
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
				onclick={() => (view = 'tracks')}>路径记录</button
			>
			<button
				class="tab"
				role="tab"
				id="tab-tokens"
				aria-selected={view === 'tokens'}
				aria-controls="panel-tokens"
				tabindex={view === 'tokens' ? 0 : -1}
				onclick={() => (view = 'tokens')}>上传设置</button
			>
		</div>
		<main>
			{#if error}<p class="notice error" role="alert">{error}</p>{/if}
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
							><span>日期范围</span><select
								class="select"
								value={preset}
								onchange={(event) => changePreset(event.currentTarget.value)}
								><option value="today">今天</option><option value="yesterday">昨天</option><option
									value="week">最近7天</option
								><option value="month">最近30天</option><option value="custom">自定义</option
								></select
							></label
						>
						<label class="field"
							><span>开始日期</span><input
								class="input"
								type="date"
								bind:value={from}
								required
								onchange={() => (preset = 'custom')}
							/></label
						>
						<label class="field"
							><span>结束日期</span><input
								class="input"
								type="date"
								bind:value={to}
								min={from}
								required
								onchange={() => (preset = 'custom')}
							/></label
						>
						<label class="field"
							><span>上报 ID</span><select
								class="select"
								bind:value={id}
								onchange={() => (device = '')}
								><option value="">全部 ID</option>{#each ids as sourceID (sourceID)}<option
										value={sourceID}>{sourceID}</option
									>{/each}</select
							></label
						>
						<label class="field"
							><span>设备</span><select class="select device-select" bind:value={device}
								><option value="">全部设备</option>{#each devices as hash (hash)}<option
										value={hash}>{hash}</option
									>{/each}</select
							></label
						>
						<button class="btn btn-primary" disabled={loading}
							>{loading ? '查询中…' : '查看路径'}</button
						>
					</form>
					<div class="summary" aria-live="polite">
						<div>
							<span class="muted">路径</span><strong>{data?.summary.track_count ?? '—'}</strong>
						</div>
						<div>
							<span class="muted">距离总和</span><strong
								>{data ? formatDistance(data.summary.distance_m) : '—'}</strong
							>
						</div>
						<div>
							<span class="muted">记录时长</span><strong
								>{data ? formatDuration(data.summary.duration_ms) : '—'}</strong
							>
						</div>
						<p class="muted small">
							时区 {state.timezone}<br />停报超过30分钟分段，时长包含段内停留
						</p>
					</div>
					<div class="workspace">
						<section class="track-list" aria-label="路径列表" aria-busy={loading}>
							<div class="list-heading">
								<h2>路径列表</h2>
								<span class="muted small">{data?.summary.point_count ?? 0} 个定位点</span>
							</div>
							{#if loading}<div class="empty" role="status">正在读取路径…</div>
							{:else if !data}<div class="empty">
									<p>路径加载失败，请重新查询。</p>
									<button class="btn btn-sm" onclick={() => reload()}>重试</button>
								</div>
							{:else if !data.tracks.length}<div class="empty">
									<h3>这个日期范围还没有路径</h3>
									<p class="muted">尝试其他日期，或前往上传设置登记手机 token。</p>
									<button class="btn btn-sm" onclick={() => (view = 'tokens')}>打开上传设置</button>
								</div>
							{:else}
								<div class="track-rows">
									{#each data.tracks as track (track.key)}<button
											class="track-row"
											class:selected={selectedKey === track.key}
											aria-pressed={selectedKey === track.key}
											onclick={() => choose(track)}
											><span class="track-id">{track.id}</span><span class="muted small"
												>{formatTime(track.start_ms, state.timezone)} — {formatTime(
													track.end_ms,
													state.timezone
												)}</span
											><span class="track-measures"
												><span>{formatDistance(track.distance_m)}</span><span
													>{formatDuration(track.duration_ms)}</span
												><span>{track.point_count} 点</span></span
											><span class="muted small device-hash">设备 {track.device_hash}</span></button
										>{/each}
								</div>
								<div class="pagination">
									<button
										class="btn btn-sm btn-ghost"
										disabled={page <= 1 || loading}
										onclick={() => reload(page - 1)}>上一页</button
									><span class="small"
										>{page} / {Math.max(1, Math.ceil(data.summary.track_count / 50))}</span
									><button
										class="btn btn-sm btn-ghost"
										disabled={page * 50 >= data.summary.track_count || loading}
										onclick={() => reload(page + 1)}>下一页</button
									>
								</div>
							{/if}
						</section>
						<section class="map-panel" aria-label="路径详情">
							<div class="map-heading">
								<h2>{selected ? selected.id : detailLoading ? '正在读取路径…' : '路径地图'}</h2>
								<span class="muted small"
									>{selected
										? `${selected.point_count} 点 · ${formatDistance(selected.distance_m)}`
										: '选择路径查看详情'}</span
								>
							</div>
							<TrackMap track={selected} timezone={state.timezone} />
							{#if selected}<details class="point-details">
									<summary>定位点详情（{selected.point_count} 点）</summary>
									<div class="point-table-scroll">
										<table class="table table-sm">
											<thead
												><tr
													><th>时间</th><th>纬度 / 经度</th><th>精度</th><th>速度</th><th>海拔</th
													><th>方向</th></tr
												></thead
											><tbody
												>{#each selected.points as point (point.timestamp_ms)}<tr
														><td>{formatTime(point.timestamp_ms, state.timezone)}</td><td
															>{point.latitude.toFixed(6)} / {point.longitude.toFixed(6)}</td
														><td>{point.accuracy_m == null ? '—' : `${point.accuracy_m} m`}</td><td
															>{point.speed_mps == null ? '—' : `${point.speed_mps} m/s`}</td
														><td>{point.altitude_m == null ? '—' : `${point.altitude_m} m`}</td><td
															>{point.bearing_deg == null ? '—' : `${point.bearing_deg}°`}</td
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
			aprs-android-hook · 距离按 GPS 定位点累计，可能受定位漂移影响
		</footer>
	</div>
{/if}
