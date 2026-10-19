// Ignore controls and gestures, including drags that return to their starting point.
export function blankClick(node, onClick) {
	let start = null;
	const down = (event) => {
		start = event.isPrimary && event.button === 0 ? event : null;
	};
	const move = (event) => {
		if (
			start &&
			event.pointerId === start.pointerId &&
			Math.hypot(event.clientX - start.clientX, event.clientY - start.clientY) >= 3
		)
			start = null;
	};
	const cancel = () => (start = null);
	const click = (event) => {
		if (start && !event.target.closest('button, a, input, select, textarea, summary')) onClick();
		start = null;
	};
	node.addEventListener('pointerdown', down);
	window.addEventListener('pointermove', move);
	window.addEventListener('pointercancel', cancel);
	node.addEventListener('scroll', cancel, true);
	node.addEventListener('click', click);
	return {
		destroy() {
			node.removeEventListener('pointerdown', down);
			window.removeEventListener('pointermove', move);
			window.removeEventListener('pointercancel', cancel);
			node.removeEventListener('scroll', cancel, true);
			node.removeEventListener('click', click);
		}
	};
}
