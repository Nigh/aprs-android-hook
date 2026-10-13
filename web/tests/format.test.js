import { test } from 'node:test';
import assert from 'node:assert/strict';
import { shiftDate, formatDistance, formatDuration, formatTime } from '../src/lib/api.js';

test('date presets cross month and year boundaries without local timezone shifts', () => {
	assert.equal(shiftDate('2026-01-01', -1), '2025-12-31');
	assert.equal(shiftDate('2024-03-01', -1), '2024-02-29');
	assert.equal(shiftDate('2026-10-13', -29), '2026-09-14');
});
test('track formatting uses recorded duration and deployment timezone', () => {
	assert.equal(formatDistance(0), '0 m');
	assert.equal(formatDistance(1234), '1.23 km');
	assert.equal(formatDuration(3600000), '1 hr 0 min');
	assert.equal(formatDuration(3600000, 'zh-CN'), '1 小时 0 分');
	assert.match(formatTime(1767225600000, 'Asia/Shanghai'), /08:00/);
	assert.match(formatTime(1767196800000, 'Asia/Shanghai', 'en'), /00:00/);
	assert.match(formatTime(1767196800000, 'Asia/Shanghai', 'zh-CN'), /00:00/);
});
