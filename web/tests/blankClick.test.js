import { test } from 'node:test';
import assert from 'node:assert/strict';
import { blankClick } from '../src/lib/blankClick.js';

test('blank clicks deselect, controls and dragging or scrolling do not', () => {
	const window = (globalThis.window = new globalThis.EventTarget());
	const node = new globalThis.EventTarget();
	let clicks = 0;
	const action = blankClick(node, () => clicks++);
	const send = (target, type, values = {}) => {
		const event = new globalThis.Event(type);
		Object.assign(event, {
			clientX: 0,
			clientY: 0,
			pointerId: 1,
			isPrimary: true,
			button: 0,
			...values
		});
		target.dispatchEvent(event);
	};
	node.closest = () => null;
	try {
		send(node, 'pointerdown');
		send(node, 'click');
		assert.equal(clicks, 1);
		send(node, 'pointerdown');
		send(window, 'pointermove', { clientX: 10 });
		send(window, 'pointermove');
		send(node, 'click');
		assert.equal(clicks, 1, 'returning to the start still counts as dragging');
		for (const type of ['scroll', 'pointercancel']) {
			send(node, 'pointerdown');
			send(type === 'scroll' ? node : window, type);
			send(node, 'click');
			assert.equal(clicks, 1);
		}
		node.closest = () => node;
		send(node, 'pointerdown');
		send(node, 'click');
		assert.equal(clicks, 1, 'controls keep their own click behavior');
		node.closest = () => null;
		send(node, 'pointerdown', { button: 2 });
		send(node, 'click');
		assert.equal(clicks, 1);
		send(node, 'pointerdown');
		send(window, 'pointermove', { clientY: 1 });
		send(node, 'click');
		assert.equal(clicks, 2, 'small pointer jitter is still a click');
		action.destroy();
		send(node, 'pointerdown');
		send(node, 'click');
		assert.equal(clicks, 2);
	} finally {
		action.destroy();
		delete globalThis.window;
	}
});
