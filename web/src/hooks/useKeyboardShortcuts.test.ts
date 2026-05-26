import { renderHook } from '@testing-library/react';
import { type ReactNode, createElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import {
	formatShortcutKeys,
	getShortcutHint,
	useKeyboardShortcuts,
} from './useKeyboardShortcuts';

function makeWrapper(initialPath = '/') {
	return function Wrapper({ children }: { children: ReactNode }) {
		return createElement(
			MemoryRouter,
			{ initialEntries: [initialPath] },
			children,
		);
	};
}

describe('useKeyboardShortcuts', () => {
	beforeEach(() => {
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	it('returns default shortcuts and helpers', () => {
		const { result } = renderHook(() => useKeyboardShortcuts(), {
			wrapper: makeWrapper(),
		});

		expect(Array.isArray(result.current.shortcuts)).toBe(true);
		expect(result.current.shortcuts.length).toBeGreaterThan(0);
		expect(result.current.defaultShortcuts.length).toBe(
			result.current.shortcuts.length,
		);
		expect(typeof result.current.updateCustomShortcut).toBe('function');
		expect(typeof result.current.resetShortcut).toBe('function');
		expect(typeof result.current.resetAllShortcuts).toBe('function');
		expect(typeof result.current.executeAction).toBe('function');
	});

	it('respects disabled option without throwing', () => {
		expect(() =>
			renderHook(() => useKeyboardShortcuts({ enabled: false }), {
				wrapper: makeWrapper(),
			}),
		).not.toThrow();
	});
});

describe('formatShortcutKeys', () => {
	it('joins keys with "then" and uppercases letters', () => {
		expect(formatShortcutKeys(['g', 'a'])).toBe('G then A');
	});

	it('formats Escape as Esc', () => {
		expect(formatShortcutKeys(['Escape'])).toBe('Esc');
	});
});

describe('getShortcutHint', () => {
	it('joins keys with + and uppercases letters', () => {
		expect(getShortcutHint(['g', 'a'])).toBe('G+A');
	});
});
