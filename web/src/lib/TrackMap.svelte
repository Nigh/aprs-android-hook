<script>
	import { onMount } from 'svelte';
	import { formatTime } from './api.js';
	import 'maplibre-gl/dist/maplibre-gl.css';
	let { track = null, timezone = 'Asia/Shanghai' } = $props();
	let container;
	let map = $state.raw(null);
	let ready = $state(false);
	let error = $state('');
	let loading = $state(true);
	let lib;
	let markers = [];
	let timeout;
	let disposed = false;
	let generation = 0;
	function fitTrack() {
		if (!ready || !map || !track?.points.length) return;
		const coordinates = track.points.map((point) => [point.longitude, point.latitude]);
		const bounds = coordinates.reduce(
			(bounds, coordinate) => bounds.extend(coordinate),
			new lib.LngLatBounds(coordinates[0], coordinates[0])
		);
		map.fitBounds(bounds, { padding: 60, maxZoom: 16, duration: 0 });
	}
	function themeColor(name) {
		const canvas = document.createElement('canvas');
		canvas.width = canvas.height = 1;
		const context = canvas.getContext('2d');
		context.fillStyle = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
		context.fillRect(0, 0, 1, 1);
		const [r, g, b] = context.getImageData(0, 0, 1, 1).data;
		return `rgb(${r},${g},${b})`;
	}
	async function initialize() {
		const current = ++generation;
		ready = false;
		loading = true;
		error = '';
		clearTimeout(timeout);
		markers.forEach((marker) => marker.remove());
		markers = [];
		map?.remove();
		map = null;
		try {
			lib = await import('maplibre-gl');
			if (disposed || current !== generation) return;
			map = new lib.Map({
				container,
				style: 'https://tiles.openfreemap.org/styles/dark',
				center: [105, 35],
				zoom: 3,
				attributionControl: { compact: true }
			});
			map.addControl(new lib.NavigationControl(), 'top-right');
			map.addControl(new lib.ScaleControl({ unit: 'metric' }));
			map.on('error', () => {
				if (current !== generation) return;
				error = '地图资源加载失败，请检查网络后重试。路径与统计仍可查看。';
				loading = false;
			});
			map.on('load', () => {
				if (current !== generation) return;
				clearTimeout(timeout);
				map.addSource('track', {
					type: 'geojson',
					data: { type: 'FeatureCollection', features: [] }
				});
				map.addLayer({
					id: 'track-line',
					type: 'line',
					source: 'track',
					layout: { 'line-cap': 'round', 'line-join': 'round' },
					paint: { 'line-color': themeColor('--color-primary'), 'line-width': 4 }
				});
				map.addLayer({
					id: 'track-points',
					type: 'circle',
					source: 'track',
					filter: ['==', '$type', 'Point'],
					paint: {
						'circle-radius': 4,
						'circle-color': themeColor('--color-accent'),
						'circle-stroke-width': 1,
						'circle-stroke-color': themeColor('--color-base-100')
					}
				});
				map.on('click', 'track-points', (event) => {
					const index = event.features?.[0]?.properties?.index;
					const point = track?.points[index];
					if (!point) return;
					const content = document.createElement('div');
					content.className = 'point-popup';
					const lines = [
						formatTime(point.timestamp_ms, timezone),
						`${point.latitude.toFixed(6)}, ${point.longitude.toFixed(6)}`,
						`精度：${point.accuracy_m == null ? '未知' : `${point.accuracy_m} m`}`,
						`速度：${point.speed_mps == null ? '未知' : `${point.speed_mps} m/s`}`,
						`海拔：${point.altitude_m == null ? '未知' : `${point.altitude_m} m`}`,
						`方向：${point.bearing_deg == null ? '未知' : `${point.bearing_deg}°`}`
					];
					lines.forEach((line) => {
						const item = document.createElement('div');
						item.textContent = line;
						content.append(item);
					});
					new lib.Popup()
						.setLngLat([point.longitude, point.latitude])
						.setDOMContent(content)
						.addTo(map);
				});
				ready = true;
				loading = false;
			});
			timeout = setTimeout(() => {
				if (!ready) {
					loading = false;
					error = '地图加载超时，请检查网络后重试。';
				}
			}, 15000);
		} catch {
			if (disposed || current !== generation) return;
			loading = false;
			error = '当前浏览器无法加载地图，请尝试支持 WebGL 的浏览器。';
		}
	}
	onMount(() => {
		initialize();
		const observer = new window.ResizeObserver(() => {
			map?.resize();
			fitTrack();
		});
		observer.observe(container);
		return () => {
			observer.disconnect();
			disposed = true;
			generation++;
			clearTimeout(timeout);
			markers.forEach((marker) => marker.remove());
			map?.remove();
		};
	});
	$effect(() => {
		if (!ready || !map) return;
		markers.forEach((marker) => marker.remove());
		markers = [];
		const points = track?.points || [];
		const coordinates = points.map((point) => [point.longitude, point.latitude]);
		const features = points.map((point, index) => ({
			type: 'Feature',
			properties: { index },
			geometry: { type: 'Point', coordinates: [point.longitude, point.latitude] }
		}));
		if (coordinates.length > 1)
			features.push({
				type: 'Feature',
				properties: {},
				geometry: { type: 'LineString', coordinates }
			});
		map.getSource('track').setData({ type: 'FeatureCollection', features });
		if (!coordinates.length) return;
		fitTrack();
		[0, points.length - 1]
			.filter((value, index, all) => all.indexOf(value) === index)
			.forEach((index) => {
				const element = document.createElement('div');
				element.className = `endpoint ${index === 0 ? 'start' : 'end'}`;
				element.textContent = index === 0 ? '起' : '终';
				element.setAttribute('aria-label', index === 0 ? '路径起点' : '路径终点');
				markers.push(new lib.Marker({ element }).setLngLat(coordinates[index]).addTo(map));
			});
	});
</script>

<div class="map-shell">
	<div class="map-canvas" bind:this={container} aria-label="路径地图"></div>
	{#if loading}<div class="map-notice" role="status">正在加载地图…</div>{/if}
	{#if error}<div class="map-notice map-error" role="status">
			<span>{error}</span><button class="btn btn-sm" onclick={initialize}>重试地图</button>
		</div>{/if}
</div>
