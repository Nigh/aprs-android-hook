<script>
	import { onMount, untrack } from 'svelte';
	import { locale, t } from '$lib/i18n.js';
	import { defaultLocale, locales, translate } from '$lib/locales/index.js';
	import { formatMeasurement, formatTime } from './api.js';
	import 'maplibre-gl/dist/maplibre-gl.css';
	let { track = null, paths = [], onSelect = () => {}, timezone = 'Asia/Shanghai' } = $props();
	let container = $state();
	let map = $state.raw(null);
	let ready = $state(false);
	let error = $state('');
	let loading = $state(true);
	let lib;
	let markers = [];
	let timeout;
	let disposed = false;
	let generation = 0;
	let hoveredPoint = null;
	let popup;
	function hoverPoint(index = null) {
		if (hoveredPoint === index) return;
		if (hoveredPoint !== null)
			map.setFeatureState({ source: 'track', id: hoveredPoint }, { hover: false });
		hoveredPoint = index;
		if (index !== null) map.setFeatureState({ source: 'track', id: index }, { hover: true });
		map.getCanvas().style.cursor = index === null ? '' : 'pointer';
	}
	function fitRange() {
		if (!ready || !map) return;
		const routes = paths.length
			? paths.map((path) => path.coordinates)
			: track?.points?.length
				? [track.points.map((point) => [point.longitude, point.latitude])]
				: [];
		if (!routes.length) return;
		const first = routes[0][0];
		const bounds = new lib.LngLatBounds(first, first);
		for (const route of routes) for (const coordinate of route) bounds.extend(coordinate);
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
	async function initialize(language) {
		const current = ++generation;
		ready = false;
		loading = true;
		error = '';
		clearTimeout(timeout);
		markers.forEach((marker) => marker.remove());
		markers = [];
		popup?.remove();
		hoveredPoint = null;
		map?.remove();
		map = null;
		try {
			lib = await import('maplibre-gl');
			if (disposed || current !== generation) return;
			map = new lib.Map({
				locale: Object.fromEntries(
					Object.keys(locales[defaultLocale].messages)
						.filter((key) => key.startsWith('maplibre.'))
						.map((key) => [key.slice('maplibre.'.length), translate(language, key)])
				),
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
				error = 'error.mapResources';
				loading = false;
			});
			map.on('load', () => {
				if (current !== generation) return;
				clearTimeout(timeout);
				map.addSource('overview', {
					type: 'geojson',
					data: { type: 'FeatureCollection', features: [] }
				});
				map.addLayer({
					id: 'overview-line',
					type: 'line',
					source: 'overview',
					layout: { 'line-cap': 'round', 'line-join': 'round' },
					paint: {
						'line-color': themeColor('--color-primary'),
						'line-width': 1.5,
						'line-opacity': 0.55
					}
				});
				map.addLayer({
					id: 'overview-hit',
					type: 'line',
					source: 'overview',
					paint: {
						'line-color': themeColor('--color-primary'),
						'line-width': 12,
						'line-opacity': 0.01
					}
				});
				map.on('mousemove', 'overview-hit', () => {
					map.getCanvas().style.cursor = 'pointer';
				});
				map.on('mouseleave', 'overview-hit', () => {
					if (hoveredPoint === null) map.getCanvas().style.cursor = '';
				});
				map.on('click', 'overview-hit', (event) => {
					const key = event.features?.[0]?.properties?.key;
					if (key && key !== track?.key) onSelect(key, event.features[0].properties.page);
				});
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
						'circle-radius': ['case', ['boolean', ['feature-state', 'hover'], false], 7, 4],
						'circle-color': themeColor('--color-base-content'),
						'circle-stroke-width': ['case', ['boolean', ['feature-state', 'hover'], false], 2, 1],
						'circle-stroke-color': themeColor('--color-base-100')
					}
				});
				map.on('mousemove', 'track-points', (event) => {
					hoverPoint(event.features?.[0]?.id ?? null);
				});
				map.on('mouseleave', 'track-points', () => hoverPoint());
				map.on('click', 'track-points', (event) => {
					const index = event.features?.[0]?.properties?.index;
					const point = track?.points[index];
					if (!point) return;
					const content = document.createElement('div');
					content.className = 'point-popup';
					const time = document.createElement('strong');
					time.textContent = formatTime(point.timestamp_ms, timezone, $locale);
					const coordinates = document.createElement('p');
					coordinates.className = 'muted';
					coordinates.textContent = `${point.latitude.toFixed(6)}, ${point.longitude.toFixed(6)}`;
					const measurements = document.createElement('dl');
					[
						['point.accuracy', point.accuracy_m, 'm'],
						['point.speed', point.speed_mps, 'm/s'],
						['point.altitude', point.altitude_m, 'm'],
						['point.bearing', point.bearing_deg, '°']
					].forEach(([label, value, unit]) => {
						const term = document.createElement('dt');
						term.textContent = $t(label);
						const detail = document.createElement('dd');
						detail.textContent = formatMeasurement(value, unit, $locale, $t('point.unknown'));
						measurements.append(term, detail);
					});
					content.append(time, coordinates, measurements);
					popup?.remove();
					popup = new lib.Popup({ maxWidth: '300px', offset: 10 })
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
					error = 'error.mapTimeout';
				}
			}, 15000);
		} catch {
			if (disposed || current !== generation) return;
			loading = false;
			error = 'error.mapBrowser';
		}
	}
	onMount(() => {
		const observer = new window.ResizeObserver(() => {
			map?.resize();
			fitRange();
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
		const language = $locale;
		if (container) untrack(() => initialize(language));
	});
	$effect(() => {
		if (!ready || !map) return;
		const features = paths.map((path) => ({
			type: 'Feature',
			properties: { key: path.key, page: path.page },
			geometry: { type: 'LineString', coordinates: path.coordinates }
		}));
		map.getSource('overview').setData({ type: 'FeatureCollection', features });
		fitRange();
	});
	$effect(() => {
		if (!ready || !map) return;
		markers.forEach((marker) => marker.remove());
		markers = [];
		popup?.remove();
		hoverPoint();
		map.removeFeatureState({ source: 'track' });
		const points = track?.points || [];
		const coordinates = points.map((point) => [point.longitude, point.latitude]);
		const features = points.map((point, index) => ({
			type: 'Feature',
			id: index,
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
		if (!paths.length) fitRange();
		[0, points.length - 1]
			.filter((value, index, all) => all.indexOf(value) === index)
			.forEach((index) => {
				const element = document.createElement('div');
				element.className = `endpoint ${index === 0 ? 'start' : 'end'}`;
				element.textContent = index === 0 ? $t('map.startShort') : $t('map.endShort');
				element.setAttribute('aria-label', index === 0 ? $t('map.start') : $t('map.end'));
				markers.push(new lib.Marker({ element }).setLngLat(coordinates[index]).addTo(map));
			});
	});
</script>

<div class="map-shell">
	<div class="map-canvas" bind:this={container} aria-label={$t('map.title')}></div>
	{#if loading}<div class="map-notice" role="status">{$t('map.loading')}</div>{/if}
	{#if error}<div class="map-notice map-error" role="status">
			<span>{$t(error)}</span><button class="btn btn-sm" onclick={() => initialize($locale)}
				>{$t('map.retry')}</button
			>
		</div>{/if}
</div>
