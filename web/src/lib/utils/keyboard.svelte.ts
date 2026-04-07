/**
 * Keyboard shortcut manager for vim-style navigation.
 *
 * Safety: shortcuts only fire when no input/textarea/select is focused.
 * View-specific shortcuts are registered/unregistered when views mount/unmount.
 */

export type KeyHandler = (e: KeyboardEvent) => void;

export interface Shortcut {
	key: string;
	handler: KeyHandler;
	/** Optional description for the help overlay. */
	description?: string;
}

/** Check whether the event target is an editable element. */
export function isEditableTarget(e: KeyboardEvent): boolean {
	const target = e.target as HTMLElement;
	if (!target) return false;
	const tag = target.tagName;
	if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
	if (target.isContentEditable) return true;
	// bits-ui select triggers
	if (target.closest('[data-slot="select-trigger"]')) return true;
	return false;
}

/**
 * A simple flash-message store for keyboard action confirmations.
 * No external toast library needed — just a reactive string.
 */
let _toast = $state<string | null>(null);
let _toastTimer: ReturnType<typeof setTimeout> | null = null;

export function showToast(message: string, durationMs = 2000): void {
	if (_toastTimer) clearTimeout(_toastTimer);
	_toast = message;
	_toastTimer = setTimeout(() => {
		_toast = null;
		_toastTimer = null;
	}, durationMs);
}

export function getToast(): string | null {
	return _toast;
}

export function clearToast(): void {
	if (_toastTimer) clearTimeout(_toastTimer);
	_toast = null;
	_toastTimer = null;
}
