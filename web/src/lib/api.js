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
		throw new Error('无法连接服务，请检查网络后重试', { cause: error });
	}
	const value = await response.json();
	if (!response.ok) {
		if (response.status === 401 && !['login', 'register'].includes(path)) {
			window.dispatchEvent(new Event('session-expired'));
		}
		throw new Error(value.error || '请求失败，请重试');
	}
	return value;
}

export function formatDistance(meters) {
	return meters >= 1000 ? `${(meters / 1000).toFixed(2)} km` : `${Math.round(meters)} m`;
}
export function formatDuration(ms) {
	const minutes = Math.floor(ms / 60000);
	return minutes >= 60 ? `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分` : `${minutes} 分钟`;
}
export function formatTime(ms, timezone) {
	return new Intl.DateTimeFormat('zh-CN', {
		timeZone: timezone,
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	}).format(ms);
}

export function shiftDate(date, days) {
	const d = new Date(`${date}T12:00:00Z`);
	d.setUTCDate(d.getUTCDate() + days);
	return d.toISOString().slice(0, 10);
}
