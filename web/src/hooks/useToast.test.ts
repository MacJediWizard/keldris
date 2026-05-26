import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useToastValue } from './useToast';

describe('useToastValue', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('starts with empty toasts array', () => {
		const { result } = renderHook(() => useToastValue());
		expect(result.current.toasts).toEqual([]);
	});

	it('addToast appends a toast and returns id', () => {
		const { result } = renderHook(() => useToastValue());
		let id = '';
		act(() => {
			id = result.current.addToast('Hello', 'info', 0);
		});
		expect(result.current.toasts).toHaveLength(1);
		expect(result.current.toasts[0].id).toBe(id);
		expect(result.current.toasts[0].message).toBe('Hello');
	});

	it('removeToast removes by id', () => {
		const { result } = renderHook(() => useToastValue());
		let id = '';
		act(() => {
			id = result.current.addToast('To remove', 'info', 0);
		});
		act(() => {
			result.current.removeToast(id);
		});
		expect(result.current.toasts).toEqual([]);
	});

	it('auto-removes after duration', () => {
		const { result } = renderHook(() => useToastValue());
		act(() => {
			result.current.addToast('Bye', 'info', 1000);
		});
		expect(result.current.toasts).toHaveLength(1);
		act(() => {
			vi.advanceTimersByTime(1100);
		});
		expect(result.current.toasts).toEqual([]);
	});

	it('success creates info variant=success toast', () => {
		const { result } = renderHook(() => useToastValue());
		act(() => {
			result.current.success('Saved!', 0);
		});
		expect(result.current.toasts[0].variant).toBe('success');
	});

	it('error creates variant=error toast', () => {
		const { result } = renderHook(() => useToastValue());
		act(() => {
			result.current.error('Boom', 0);
		});
		expect(result.current.toasts[0].variant).toBe('error');
	});

	it('warning creates variant=warning toast', () => {
		const { result } = renderHook(() => useToastValue());
		act(() => {
			result.current.warning('Heads up', 0);
		});
		expect(result.current.toasts[0].variant).toBe('warning');
	});

	it('info creates variant=info toast', () => {
		const { result } = renderHook(() => useToastValue());
		act(() => {
			result.current.info('FYI', 0);
		});
		expect(result.current.toasts[0].variant).toBe('info');
	});
});
